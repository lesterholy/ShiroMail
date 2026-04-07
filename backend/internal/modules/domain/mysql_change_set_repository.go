package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

func (r *MySQLRepository) SaveDNSChangeSet(ctx context.Context, item DNSChangeSet) (DNSChangeSet, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return DNSChangeSet{}, tx.Error
	}

	zoneID, err := r.ensureDNSZoneRow(ctx, tx, item)
	if err != nil {
		tx.Rollback()
		return DNSChangeSet{}, err
	}
	item.ZoneID = zoneID

	changeSetRow := database.DNSChangeSetRow{
		ID:                  item.ID,
		ZoneID:              item.ZoneID,
		ProviderAccountID:   item.ProviderAccountID,
		ProviderZoneID:      item.ProviderZoneID,
		ZoneName:            item.ZoneName,
		RequestedByUserID:   item.RequestedByUserID,
		RequestedByAPIKeyID: item.RequestedByAPIKeyID,
		Status:              item.Status,
		Provider:            item.Provider,
		Summary:             item.Summary,
		AppliedAt:           item.AppliedAt,
	}

	if item.ID == 0 {
		if err := tx.Create(&changeSetRow).Error; err != nil {
			tx.Rollback()
			return DNSChangeSet{}, err
		}
	} else {
		var existing database.DNSChangeSetRow
		if err := tx.First(&existing, "id = ?", item.ID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return DNSChangeSet{}, ErrDNSChangeSetNotFound
			}
			return DNSChangeSet{}, err
		}
		if err := tx.Model(&database.DNSChangeSetRow{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"zone_id":                 changeSetRow.ZoneID,
				"provider_account_id":     changeSetRow.ProviderAccountID,
				"provider_zone_id":        changeSetRow.ProviderZoneID,
				"zone_name":               changeSetRow.ZoneName,
				"requested_by_user_id":    changeSetRow.RequestedByUserID,
				"requested_by_api_key_id": changeSetRow.RequestedByAPIKeyID,
				"status":                  changeSetRow.Status,
				"provider":                changeSetRow.Provider,
				"summary":                 changeSetRow.Summary,
				"applied_at":              changeSetRow.AppliedAt,
			}).Error; err != nil {
			tx.Rollback()
			return DNSChangeSet{}, err
		}
		if err := tx.Where("change_set_id = ?", item.ID).Delete(&database.DNSChangeOperationRow{}).Error; err != nil {
			tx.Rollback()
			return DNSChangeSet{}, err
		}
	}

	for _, operation := range item.Operations {
		beforeJSON, err := marshalProviderRecord(operation.Before)
		if err != nil {
			tx.Rollback()
			return DNSChangeSet{}, err
		}
		afterJSON, err := marshalProviderRecord(operation.After)
		if err != nil {
			tx.Rollback()
			return DNSChangeSet{}, err
		}
		operationRow := database.DNSChangeOperationRow{
			ChangeSetID: changeSetRow.ID,
			Operation:   operation.Operation,
			RecordType:  operation.RecordType,
			RecordName:  operation.RecordName,
			BeforeJSON:  beforeJSON,
			AfterJSON:   afterJSON,
			Status:      operation.Status,
		}
		if err := tx.Create(&operationRow).Error; err != nil {
			tx.Rollback()
			return DNSChangeSet{}, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return DNSChangeSet{}, err
	}
	return r.GetDNSChangeSetByID(ctx, changeSetRow.ID)
}

func (r *MySQLRepository) ListDNSChangeSets(ctx context.Context, providerAccountID uint64, providerZoneID string) ([]DNSChangeSet, error) {
	var rows []database.DNSChangeSetRow
	query := r.db.WithContext(ctx).
		Where("provider_account_id = ?", providerAccountID).
		Order("created_at DESC").
		Order("id DESC")
	if strings.TrimSpace(providerZoneID) != "" {
		query = query.Where("provider_zone_id = ?", strings.TrimSpace(providerZoneID))
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]DNSChangeSet, 0, len(rows))
	for _, row := range rows {
		item, err := r.GetDNSChangeSetByID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *MySQLRepository) GetDNSChangeSetByID(ctx context.Context, id uint64) (DNSChangeSet, error) {
	var row database.DNSChangeSetRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DNSChangeSet{}, ErrDNSChangeSetNotFound
		}
		return DNSChangeSet{}, err
	}

	var operationRows []database.DNSChangeOperationRow
	if err := r.db.WithContext(ctx).
		Where("change_set_id = ?", row.ID).
		Order("id ASC").
		Find(&operationRows).Error; err != nil {
		return DNSChangeSet{}, err
	}

	operations := make([]DNSChangeOperation, 0, len(operationRows))
	for _, operationRow := range operationRows {
		before, err := unmarshalProviderRecord(operationRow.BeforeJSON)
		if err != nil {
			return DNSChangeSet{}, err
		}
		after, err := unmarshalProviderRecord(operationRow.AfterJSON)
		if err != nil {
			return DNSChangeSet{}, err
		}
		operations = append(operations, DNSChangeOperation{
			ID:          operationRow.ID,
			ChangeSetID: operationRow.ChangeSetID,
			Operation:   operationRow.Operation,
			RecordType:  operationRow.RecordType,
			RecordName:  operationRow.RecordName,
			Before:      before,
			After:       after,
			Status:      operationRow.Status,
		})
	}

	return DNSChangeSet{
		ID:                  row.ID,
		ZoneID:              row.ZoneID,
		ProviderAccountID:   row.ProviderAccountID,
		ProviderZoneID:      row.ProviderZoneID,
		ZoneName:            row.ZoneName,
		RequestedByUserID:   row.RequestedByUserID,
		RequestedByAPIKeyID: row.RequestedByAPIKeyID,
		Status:              row.Status,
		Provider:            row.Provider,
		Summary:             row.Summary,
		Operations:          operations,
		CreatedAt:           row.CreatedAt,
		AppliedAt:           row.AppliedAt,
	}, nil
}

func (r *MySQLRepository) ensureDNSZoneRow(ctx context.Context, tx *gorm.DB, item DNSChangeSet) (*uint64, error) {
	if item.ZoneID != nil {
		return item.ZoneID, nil
	}

	zoneName := strings.TrimSpace(item.ZoneName)
	providerZoneID := strings.TrimSpace(item.ProviderZoneID)
	if zoneName == "" && providerZoneID == "" {
		return nil, nil
	}

	var row database.DNSZoneRow
	query := tx.WithContext(ctx)
	switch {
	case providerZoneID != "" && item.ProviderAccountID != 0:
		err := query.Where("provider_account_id = ? AND provider_zone_id = ?", item.ProviderAccountID, providerZoneID).First(&row).Error
		if err == nil {
			return &row.ID, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if zoneName != "" {
		err := query.Where("zone_name = ?", zoneName).First(&row).Error
		if err == nil {
			updates := map[string]any{}
			if providerZoneID != "" && row.ProviderZoneID != providerZoneID {
				updates["provider_zone_id"] = providerZoneID
			}
			if item.ProviderAccountID != 0 && (row.ProviderAccountID == nil || *row.ProviderAccountID != item.ProviderAccountID) {
				updates["provider_account_id"] = item.ProviderAccountID
			}
			if len(updates) != 0 {
				if err := query.Model(&database.DNSZoneRow{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
					return nil, err
				}
			}
			return &row.ID, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	create := database.DNSZoneRow{
		ProviderZoneID:    providerZoneID,
		ZoneName:          zoneName,
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		HealthStatus:      "unknown",
	}
	if item.ProviderAccountID != 0 {
		create.ProviderAccountID = &item.ProviderAccountID
	}
	if create.ZoneName == "" {
		create.ZoneName = providerZoneID
	}
	if create.ProviderZoneID == "" {
		create.ProviderZoneID = create.ZoneName
	}
	if err := tx.WithContext(ctx).Create(&create).Error; err != nil {
		return nil, err
	}
	return &create.ID, nil
}

func marshalProviderRecord(record *ProviderRecord) ([]byte, error) {
	if record == nil {
		return nil, nil
	}
	return json.Marshal(record)
}

func unmarshalProviderRecord(payload []byte) (*ProviderRecord, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var record ProviderRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	return &record, nil
}
