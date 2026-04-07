package extractor

import (
	"context"
	"errors"
)

var ErrRuleNotFound = errors.New("extractor rule not found")

type Repository interface {
	ListUserRules(ctx context.Context, userID uint64) ([]Rule, error)
	ListAdminTemplates(ctx context.Context) ([]Rule, error)
	CreateRule(ctx context.Context, rule Rule) (Rule, error)
	UpdateRule(ctx context.Context, rule Rule) (Rule, error)
	FindRuleByID(ctx context.Context, ruleID uint64) (Rule, error)
	DeleteRule(ctx context.Context, ruleID uint64) error
	ListEnabledTemplateIDs(ctx context.Context, userID uint64) ([]uint64, error)
	SetTemplateEnabled(ctx context.Context, userID uint64, ruleID uint64, enabled bool) error
}
