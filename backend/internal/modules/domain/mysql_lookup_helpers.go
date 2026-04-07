package domain

import (
	"context"

	"gorm.io/gorm"
)

func findFirstRow[T any](ctx context.Context, db *gorm.DB, row *T, query string, args ...any) (bool, error) {
	result := db.WithContext(ctx).Where(query, args...).Limit(1).Find(row)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
