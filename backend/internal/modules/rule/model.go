package rule

import (
	"context"
	"sort"
	"sync"
	"time"
)

type Rule struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	RetentionHours int       `json:"retentionHours"`
	AutoExtend     bool      `json:"autoExtend"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Repository interface {
	List(ctx context.Context) ([]Rule, error)
	Upsert(ctx context.Context, item Rule) (Rule, error)
}

type MemoryRepository struct {
	mu      sync.RWMutex
	records map[string]Rule
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		records: map[string]Rule{},
	}
}

func (r *MemoryRepository) List(_ context.Context) ([]Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Rule, 0, len(r.records))
	for _, item := range r.records {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (r *MemoryRepository) Upsert(_ context.Context, item Rule) (Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item.UpdatedAt = time.Now()
	r.records[item.ID] = item
	return item, nil
}
