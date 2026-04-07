package system

import (
	"context"
	"time"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

type MySQLJobRepository struct {
	db *gorm.DB
}

func NewMySQLJobRepository(db *gorm.DB) *MySQLJobRepository {
	return &MySQLJobRepository{db: db}
}

func (r *MySQLJobRepository) List(ctx context.Context) ([]JobRecord, error) {
	var rows []database.JobRow
	if err := r.db.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]JobRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapJobRow(row))
	}
	return items, nil
}

func (r *MySQLJobRepository) Create(ctx context.Context, jobType string, status string, errorMessage string) (JobRecord, error) {
	row := database.JobRow{
		JobType:      jobType,
		Status:       status,
		Payload:      []byte(`{}`),
		ErrorMessage: errorMessage,
		ScheduledAt:  time.Now(),
		CreatedAt:    time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return JobRecord{}, err
	}
	return mapJobRow(row), nil
}

func (r *MySQLJobRepository) CountFailed(ctx context.Context) int {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&database.JobRow{}).
		Where("status = ?", "failed").
		Count(&count).Error; err != nil {
		return 0
	}
	return int(count)
}

func mapJobRow(row database.JobRow) JobRecord {
	return JobRecord{
		ID:           row.ID,
		JobType:      row.JobType,
		Status:       row.Status,
		ErrorMessage: row.ErrorMessage,
		CreatedAt:    row.CreatedAt,
	}
}
