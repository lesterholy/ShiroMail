package rule

import (
	"context"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) List(ctx context.Context) ([]Rule, error) {
	var rows []database.RuleRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]Rule, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapRuleRow(row))
	}
	return items, nil
}

func (r *MySQLRepository) Upsert(ctx context.Context, item Rule) (Rule, error) {
	row := database.RuleRow{
		ID:             item.ID,
		Name:           item.Name,
		RetentionHours: item.RetentionHours,
		AutoExtend:     item.AutoExtend,
	}

	if err := r.db.WithContext(ctx).
		Where("id = ?", item.ID).
		Assign(database.RuleRow{
			Name:           row.Name,
			RetentionHours: row.RetentionHours,
			AutoExtend:     row.AutoExtend,
		}).
		FirstOrCreate(&row).Error; err != nil {
		return Rule{}, err
	}

	var persisted database.RuleRow
	if err := r.db.WithContext(ctx).First(&persisted, "id = ?", item.ID).Error; err != nil {
		return Rule{}, err
	}
	return mapRuleRow(persisted), nil
}

func mapRuleRow(row database.RuleRow) Rule {
	return Rule{
		ID:             row.ID,
		Name:           row.Name,
		RetentionHours: row.RetentionHours,
		AutoExtend:     row.AutoExtend,
		UpdatedAt:      row.UpdatedAt,
	}
}
