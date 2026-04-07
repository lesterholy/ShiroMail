package system

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

type MySQLConfigRepository struct {
	db *gorm.DB
}

func NewMySQLConfigRepository(db *gorm.DB) *MySQLConfigRepository {
	return &MySQLConfigRepository{db: db}
}

func (r *MySQLConfigRepository) List(ctx context.Context) ([]ConfigEntry, error) {
	var rows []database.SystemConfigRow
	if err := r.db.WithContext(ctx).Order("config_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]ConfigEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapConfigRow(row))
	}
	return items, nil
}

func (r *MySQLConfigRepository) Upsert(ctx context.Context, key string, value map[string]any, actorID uint64) (ConfigEntry, error) {
	body, err := json.Marshal(cloneMap(value))
	if err != nil {
		return ConfigEntry{}, err
	}

	row := database.SystemConfigRow{
		ConfigKey:   key,
		ConfigValue: body,
		UpdatedBy:   actorID,
	}
	if err := r.db.WithContext(ctx).
		Where("config_key = ?", key).
		Assign(database.SystemConfigRow{
			ConfigValue: body,
			UpdatedBy:   actorID,
		}).
		FirstOrCreate(&row).Error; err != nil {
		return ConfigEntry{}, err
	}

	var persisted database.SystemConfigRow
	if err := r.db.WithContext(ctx).First(&persisted, "config_key = ?", key).Error; err != nil {
		return ConfigEntry{}, err
	}
	return mapConfigRow(persisted), nil
}

func (r *MySQLConfigRepository) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("config_key = ?", key).Delete(&database.SystemConfigRow{}).Error
}

func mapConfigRow(row database.SystemConfigRow) ConfigEntry {
	value := map[string]any{}
	_ = json.Unmarshal(row.ConfigValue, &value)
	return ConfigEntry{
		Key:       row.ConfigKey,
		Value:     value,
		UpdatedBy: row.UpdatedBy,
		UpdatedAt: row.UpdatedAt,
	}
}
