package extractor

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu               sync.RWMutex
	nextRuleID       uint64
	rules            map[uint64]Rule
	enabledTemplates map[uint64]map[uint64]bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nextRuleID:       1,
		rules:            map[uint64]Rule{},
		enabledTemplates: map[uint64]map[uint64]bool{},
	}
}

func (r *MemoryRepository) ListUserRules(_ context.Context, userID uint64) ([]Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Rule, 0)
	for _, item := range r.rules {
		if item.OwnerUserID == nil || *item.OwnerUserID != userID {
			continue
		}
		items = append(items, cloneRule(item))
	}
	sortRules(items)
	return items, nil
}

func (r *MemoryRepository) ListAdminTemplates(_ context.Context) ([]Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Rule, 0)
	for _, item := range r.rules {
		if item.SourceType != RuleSourceAdminDefault {
			continue
		}
		items = append(items, cloneRule(item))
	}
	sortRules(items)
	return items, nil
}

func (r *MemoryRepository) CreateRule(_ context.Context, rule Rule) (Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rule.ID = r.nextRuleID
	r.nextRuleID++
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	r.rules[rule.ID] = cloneRule(rule)
	return cloneRule(rule), nil
}

func (r *MemoryRepository) UpdateRule(_ context.Context, rule Rule) (Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.rules[rule.ID]
	if !ok {
		return Rule{}, ErrRuleNotFound
	}
	rule.CreatedAt = current.CreatedAt
	rule.UpdatedAt = time.Now()
	r.rules[rule.ID] = cloneRule(rule)
	return cloneRule(rule), nil
}

func (r *MemoryRepository) FindRuleByID(_ context.Context, ruleID uint64) (Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.rules[ruleID]
	if !ok {
		return Rule{}, ErrRuleNotFound
	}
	return cloneRule(item), nil
}

func (r *MemoryRepository) DeleteRule(_ context.Context, ruleID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.rules[ruleID]; !ok {
		return ErrRuleNotFound
	}
	delete(r.rules, ruleID)
	for userID, assignments := range r.enabledTemplates {
		delete(assignments, ruleID)
		if len(assignments) == 0 {
			delete(r.enabledTemplates, userID)
		}
	}
	return nil
}

func (r *MemoryRepository) ListEnabledTemplateIDs(_ context.Context, userID uint64) ([]uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	assignments := r.enabledTemplates[userID]
	items := make([]uint64, 0, len(assignments))
	for ruleID, enabled := range assignments {
		if enabled {
			items = append(items, ruleID)
		}
	}
	slices.Sort(items)
	return items, nil
}

func (r *MemoryRepository) SetTemplateEnabled(_ context.Context, userID uint64, ruleID uint64, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.rules[ruleID]; !ok {
		return ErrRuleNotFound
	}
	assignments, ok := r.enabledTemplates[userID]
	if !ok {
		assignments = map[uint64]bool{}
		r.enabledTemplates[userID] = assignments
	}
	assignments[ruleID] = enabled
	return nil
}

func cloneRule(rule Rule) Rule {
	cloned := rule
	cloned.TargetFields = append([]TargetField(nil), rule.TargetFields...)
	cloned.MailboxIDs = append([]uint64(nil), rule.MailboxIDs...)
	cloned.DomainIDs = append([]uint64(nil), rule.DomainIDs...)
	return cloned
}

func sortRules(items []Rule) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		return items[i].ID < items[j].ID
	})
}
