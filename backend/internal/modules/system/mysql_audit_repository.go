package system

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

type MySQLAuditRepository struct {
	db *gorm.DB
}

func NewMySQLAuditRepository(db *gorm.DB) *MySQLAuditRepository {
	return &MySQLAuditRepository{db: db}
}

func (r *MySQLAuditRepository) List(ctx context.Context) ([]AuditLog, error) {
	var rows []database.AuditLogRow
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AuditLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAuditRow(row))
	}
	return items, nil
}

func (r *MySQLAuditRepository) Create(ctx context.Context, actorID uint64, action string, resourceType string, resourceID string, detail map[string]any) (AuditLog, error) {
	body, err := json.Marshal(cloneMap(detail))
	if err != nil {
		return AuditLog{}, err
	}

	row := database.AuditLogRow{
		ActorUserID:  actorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Detail:       body,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return AuditLog{}, err
	}
	return mapAuditRow(row), nil
}

func mapAuditRow(row database.AuditLogRow) AuditLog {
	detail := map[string]any{}
	_ = json.Unmarshal(row.Detail, &detail)
	return AuditLog{
		ID:           row.ID,
		ActorUserID:  row.ActorUserID,
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		Detail:       detail,
		CreatedAt:    row.CreatedAt,
	}
}
