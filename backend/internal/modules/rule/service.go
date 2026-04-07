package rule

import (
	"context"

	"shiro-email/backend/internal/modules/system"
)

type Service struct {
	repo      Repository
	auditRepo system.AuditRepository
}

func NewService(repo Repository, auditRepo system.AuditRepository) *Service {
	return &Service{
		repo:      repo,
		auditRepo: auditRepo,
	}
}

func (s *Service) List(ctx context.Context) ([]Rule, error) {
	return s.repo.List(ctx)
}

func (s *Service) Upsert(ctx context.Context, actorID uint64, item Rule) (Rule, error) {
	updated, err := s.repo.Upsert(ctx, item)
	if err != nil {
		return Rule{}, err
	}
	_, _ = s.auditRepo.Create(ctx, actorID, "admin.rule.upsert", "rule", updated.ID, map[string]any{
		"name":           updated.Name,
		"retentionHours": updated.RetentionHours,
		"autoExtend":     updated.AutoExtend,
	})
	return updated, nil
}
