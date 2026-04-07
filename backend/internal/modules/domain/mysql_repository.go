package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) ListAll(ctx context.Context) ([]Domain, error) {
	var rows []database.DomainRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.hydrateDomains(ctx, rows)
}

func (r *MySQLRepository) ListActive(ctx context.Context) ([]Domain, error) {
	var rows []database.DomainRow
	if err := r.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return r.hydrateDomains(ctx, rows)
}

func (r *MySQLRepository) ListAccessibleActive(ctx context.Context, userID uint64) ([]Domain, error) {
	items, err := r.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	accessible := make([]Domain, 0, len(items))
	for _, item := range items {
		if isAccessibleToUser(item, userID) {
			accessible = append(accessible, item)
		}
	}
	return accessible, nil
}

func (r *MySQLRepository) GetActiveByID(ctx context.Context, id uint64) (Domain, error) {
	var row database.DomainRow
	if err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, "active").
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Domain{}, ErrDomainNotFound
		}
		return Domain{}, err
	}
	items, err := r.hydrateDomains(ctx, []database.DomainRow{row})
	if err != nil {
		return Domain{}, err
	}
	return items[0], nil
}

func (r *MySQLRepository) GetAccessibleActiveByID(ctx context.Context, userID uint64, id uint64) (Domain, error) {
	item, err := r.GetActiveByID(ctx, id)
	if err != nil {
		return Domain{}, err
	}
	if !isAccessibleToUser(item, userID) {
		return Domain{}, ErrDomainNotFound
	}
	return item, nil
}

func (r *MySQLRepository) FindByDomain(ctx context.Context, hostname string) (Domain, error) {
	var row database.DomainRow
	if err := r.db.WithContext(ctx).
		Where("domain = ?", hostname).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Domain{}, ErrDomainNotFound
		}
		return Domain{}, err
	}
	items, err := r.hydrateDomains(ctx, []database.DomainRow{row})
	if err != nil {
		return Domain{}, err
	}
	return items[0], nil
}

func (r *MySQLRepository) Upsert(ctx context.Context, item Domain) (Domain, error) {
	item = applyDomainDefaults(item)
	row := database.DomainRow{
		ID:                item.ID,
		Domain:            item.Domain,
		Status:            item.Status,
		OwnerUserID:       item.OwnerUserID,
		Visibility:        item.Visibility,
		PublicationStatus: item.PublicationStatus,
		VerificationScore: item.VerificationScore,
		HealthStatus:      item.HealthStatus,
		IsDefault:         item.IsDefault,
		Weight:            item.Weight,
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return Domain{}, tx.Error
	}

	var existing database.DomainRow
	exists := false
	if item.ID != 0 {
		err := tx.Where("id = ?", item.ID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return Domain{}, err
		}
		exists = err == nil
	}

	if !exists {
		err := tx.Where("domain = ?", item.Domain).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return Domain{}, err
		}
		exists = err == nil
	}

	if exists {
		row.ID = existing.ID
		row.OwnerUserID = preserveOwner(existing.OwnerUserID, item.OwnerUserID)

		updates := map[string]any{
			"domain":             row.Domain,
			"status":             row.Status,
			"visibility":         row.Visibility,
			"publication_status": row.PublicationStatus,
			"verification_score": row.VerificationScore,
			"health_status":      row.HealthStatus,
			"is_default":         row.IsDefault,
			"weight":             row.Weight,
		}
		if item.OwnerUserID != nil {
			updates["owner_user_id"] = *item.OwnerUserID
		}

		if err := tx.Model(&database.DomainRow{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			tx.Rollback()
			return Domain{}, err
		}
	} else {
		if err := tx.Create(&row).Error; err != nil {
			tx.Rollback()
			return Domain{}, err
		}
	}

	if err := upsertDNSZoneBinding(ctx, tx, row, item.ProviderAccountID); err != nil {
		tx.Rollback()
		return Domain{}, err
	}
	if err := tx.Commit().Error; err != nil {
		return Domain{}, err
	}
	return r.getByID(ctx, row.ID)
}

func (r *MySQLRepository) DeleteDomain(ctx context.Context, id uint64) error {
	var row database.DomainRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDomainNotFound
		}
		return err
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if _, _, _, kind := classifyDomain(row.Domain); kind == "root" {
		if err := tx.Where("zone_name = ?", row.Domain).Delete(&database.DNSZoneRow{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Where("id = ?", id).Delete(&database.DomainRow{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *MySQLRepository) ListProviderAccounts(ctx context.Context) ([]ProviderAccount, error) {
	var rows []database.ProviderAccountRow
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ProviderAccount, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapProviderAccountRow(row))
	}
	return items, nil
}

func (r *MySQLRepository) GetProviderAccountByID(ctx context.Context, id uint64) (ProviderAccount, error) {
	var row database.ProviderAccountRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProviderAccount{}, ErrProviderAccountNotFound
		}
		return ProviderAccount{}, err
	}
	return mapProviderAccountRow(row), nil
}

func (r *MySQLRepository) CreateProviderAccount(ctx context.Context, item ProviderAccount) (ProviderAccount, error) {
	capabilitiesJSON, err := json.Marshal(item.Capabilities)
	if err != nil {
		return ProviderAccount{}, err
	}
	row := database.ProviderAccountRow{
		Provider:         item.Provider,
		OwnerType:        item.OwnerType,
		OwnerUserID:      item.OwnerUserID,
		DisplayName:      item.DisplayName,
		AuthType:         item.AuthType,
		SecretRef:        item.SecretRef,
		Status:           item.Status,
		CapabilitiesJSON: capabilitiesJSON,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return ProviderAccount{}, err
	}
	return mapProviderAccountRow(row), nil
}

func (r *MySQLRepository) UpdateProviderAccount(ctx context.Context, item ProviderAccount) (ProviderAccount, error) {
	capabilitiesJSON, err := json.Marshal(item.Capabilities)
	if err != nil {
		return ProviderAccount{}, err
	}

	var existing database.ProviderAccountRow
	if err := r.db.WithContext(ctx).First(&existing, "id = ?", item.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProviderAccount{}, ErrProviderAccountNotFound
		}
		return ProviderAccount{}, err
	}

	updates := map[string]any{
		"provider":          item.Provider,
		"owner_type":        item.OwnerType,
		"owner_user_id":     item.OwnerUserID,
		"display_name":      item.DisplayName,
		"auth_type":         item.AuthType,
		"status":            item.Status,
		"capabilities_json": capabilitiesJSON,
		"last_sync_at":      item.LastSyncAt,
	}
	if strings.TrimSpace(item.SecretRef) == "" {
		updates["secret_ref"] = existing.SecretRef
	} else {
		updates["secret_ref"] = item.SecretRef
	}

	if err := r.db.WithContext(ctx).Model(&database.ProviderAccountRow{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return ProviderAccount{}, err
	}
	return r.GetProviderAccountByID(ctx, item.ID)
}

func (r *MySQLRepository) DeleteProviderAccount(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&database.ProviderAccountRow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderAccountNotFound
	}
	return nil
}

func (r *MySQLRepository) CountActive(ctx context.Context) int {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&database.DomainRow{}).
		Where("status = ?", "active").
		Count(&count).Error; err != nil {
		return 0
	}
	return int(count)
}

func (r *MySQLRepository) getByID(ctx context.Context, id uint64) (Domain, error) {
	var row database.DomainRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Domain{}, ErrDomainNotFound
		}
		return Domain{}, err
	}
	items, err := r.hydrateDomains(ctx, []database.DomainRow{row})
	if err != nil {
		return Domain{}, err
	}
	return items[0], nil
}

func (r *MySQLRepository) hydrateDomains(ctx context.Context, rows []database.DomainRow) ([]Domain, error) {
	items := make([]Domain, 0, len(rows))
	if len(rows) == 0 {
		return items, nil
	}

	rootZones := make([]string, 0, len(rows))
	seenZones := map[string]struct{}{}
	for _, row := range rows {
		rootDomain, _, _, _ := classifyDomain(row.Domain)
		if _, ok := seenZones[rootDomain]; ok {
			continue
		}
		seenZones[rootDomain] = struct{}{}
		rootZones = append(rootZones, rootDomain)
	}

	zoneMap, providerMap, err := r.loadZoneProviderMetadata(ctx, rootZones)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		rootDomain, parentDomain, level, kind := classifyDomain(row.Domain)
		item := Domain{
			ID:                row.ID,
			Domain:            row.Domain,
			Status:            row.Status,
			OwnerUserID:       row.OwnerUserID,
			Visibility:        row.Visibility,
			PublicationStatus: row.PublicationStatus,
			VerificationScore: row.VerificationScore,
			HealthStatus:      row.HealthStatus,
			IsDefault:         row.IsDefault,
			Weight:            row.Weight,
			RootDomain:        rootDomain,
			ParentDomain:      parentDomain,
			Level:             level,
			Kind:              kind,
		}

		if zone, ok := zoneMap[rootDomain]; ok {
			if item.OwnerUserID == nil {
				item.OwnerUserID = zone.OwnerUserID
			}
			item.ProviderAccountID = zone.ProviderAccountID
			if zone.ProviderAccountID != nil {
				if provider, exists := providerMap[*zone.ProviderAccountID]; exists {
					item.Provider = provider.Provider
					item.ProviderDisplayName = provider.DisplayName
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

type zoneBinding struct {
	ProviderAccountID *uint64
	OwnerUserID       *uint64
	Visibility        string
	PublicationStatus string
	VerificationScore int
	HealthStatus      string
}

func (r *MySQLRepository) loadZoneProviderMetadata(ctx context.Context, zoneNames []string) (map[string]zoneBinding, map[uint64]ProviderAccount, error) {
	zoneMap := map[string]zoneBinding{}
	providerMap := map[uint64]ProviderAccount{}
	if len(zoneNames) == 0 {
		return zoneMap, providerMap, nil
	}

	var zones []database.DNSZoneRow
	if err := r.db.WithContext(ctx).Where("zone_name IN ?", zoneNames).Find(&zones).Error; err != nil {
		return nil, nil, err
	}

	providerIDs := make([]uint64, 0, len(zones))
	seenProviderIDs := map[uint64]struct{}{}
	for _, zone := range zones {
		zoneMap[zone.ZoneName] = zoneBinding{
			ProviderAccountID: zone.ProviderAccountID,
			OwnerUserID:       zone.OwnerUserID,
			Visibility:        zone.Visibility,
			PublicationStatus: zone.PublicationStatus,
			VerificationScore: zone.VerificationScore,
			HealthStatus:      zone.HealthStatus,
		}
		if zone.ProviderAccountID == nil {
			continue
		}
		if _, ok := seenProviderIDs[*zone.ProviderAccountID]; ok {
			continue
		}
		seenProviderIDs[*zone.ProviderAccountID] = struct{}{}
		providerIDs = append(providerIDs, *zone.ProviderAccountID)
	}

	if len(providerIDs) == 0 {
		return zoneMap, providerMap, nil
	}

	var providers []database.ProviderAccountRow
	if err := r.db.WithContext(ctx).Where("id IN ?", providerIDs).Find(&providers).Error; err != nil {
		return nil, nil, err
	}
	for _, row := range providers {
		providerMap[row.ID] = mapProviderAccountRow(row)
	}
	return zoneMap, providerMap, nil
}

func upsertDNSZoneBinding(ctx context.Context, db *gorm.DB, row database.DomainRow, providerAccountID *uint64) error {
	rootDomain, _, _, kind := classifyDomain(row.Domain)
	if kind != "root" {
		return nil
	}

	values := database.DNSZoneRow{
		ProviderZoneID:    rootDomain,
		OwnerUserID:       row.OwnerUserID,
		ZoneName:          row.Domain,
		Status:            row.Status,
		Visibility:        row.Visibility,
		PublicationStatus: row.PublicationStatus,
		VerificationScore: row.VerificationScore,
		HealthStatus:      row.HealthStatus,
	}
	if providerAccountID != nil {
		values.ProviderAccountID = providerAccountID
	}

	var existing database.DNSZoneRow
	err := db.WithContext(ctx).Where("zone_name = ?", rootDomain).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(&values).Error
	}

	updates := map[string]any{
		"status":             values.Status,
		"visibility":         values.Visibility,
		"publication_status": values.PublicationStatus,
		"verification_score": values.VerificationScore,
		"health_status":      values.HealthStatus,
		"provider_zone_id":   values.ProviderZoneID,
		"owner_user_id":      values.OwnerUserID,
	}
	if providerAccountID == nil {
		updates["provider_account_id"] = nil
	} else {
		updates["provider_account_id"] = *providerAccountID
	}
	return db.WithContext(ctx).Model(&database.DNSZoneRow{}).Where("id = ?", existing.ID).Updates(updates).Error
}

func mapProviderAccountRow(row database.ProviderAccountRow) ProviderAccount {
	var capabilities []string
	if len(row.CapabilitiesJSON) != 0 {
		_ = json.Unmarshal(row.CapabilitiesJSON, &capabilities)
	}
	return ProviderAccount{
		ID:           row.ID,
		Provider:     row.Provider,
		OwnerType:    row.OwnerType,
		OwnerUserID:  row.OwnerUserID,
		DisplayName:  row.DisplayName,
		AuthType:     row.AuthType,
		SecretRef:    row.SecretRef,
		HasSecret:    strings.TrimSpace(row.SecretRef) != "",
		Status:       row.Status,
		Capabilities: capabilities,
		LastSyncAt:   row.LastSyncAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
