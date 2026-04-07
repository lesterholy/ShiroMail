package extractor

import (
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"shiro-email/backend/internal/database"
)

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) ListUserRules(ctx context.Context, userID uint64) ([]Rule, error) {
	var rows []database.MailExtractorRuleRow
	if err := r.db.WithContext(ctx).
		Where("owner_user_id = ?", userID).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return mapRuleRows(rows)
}

func (r *MySQLRepository) ListAdminTemplates(ctx context.Context) ([]Rule, error) {
	var rows []database.MailExtractorRuleRow
	if err := r.db.WithContext(ctx).
		Where("source_type = ?", RuleSourceAdminDefault).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return mapRuleRows(rows)
}

func (r *MySQLRepository) CreateRule(ctx context.Context, rule Rule) (Rule, error) {
	row, err := mapRuleToRow(rule)
	if err != nil {
		return Rule{}, err
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return Rule{}, err
	}
	return mapRuleRow(row)
}

func (r *MySQLRepository) UpdateRule(ctx context.Context, rule Rule) (Rule, error) {
	row, err := mapRuleToRow(rule)
	if err != nil {
		return Rule{}, err
	}
	if err := r.db.WithContext(ctx).Model(&database.MailExtractorRuleRow{}).
		Where("id = ?", rule.ID).
		Updates(map[string]any{
			"owner_user_id":       row.OwnerUserID,
			"source_type":         row.SourceType,
			"template_key":        row.TemplateKey,
			"name":                row.Name,
			"description":         row.Description,
			"label":               row.Label,
			"enabled":             row.Enabled,
			"target_fields_json":  row.TargetFieldsJSON,
			"pattern":             row.Pattern,
			"flags":               row.Flags,
			"result_mode":         row.ResultMode,
			"capture_group_index": row.CaptureGroupIndex,
			"mailbox_scope_json":  row.MailboxScopeJSON,
			"domain_scope_json":   row.DomainScopeJSON,
			"sender_contains":     row.SenderContains,
			"subject_contains":    row.SubjectContains,
			"sort_order":          row.SortOrder,
		}).Error; err != nil {
		return Rule{}, err
	}
	return r.FindRuleByID(ctx, rule.ID)
}

func (r *MySQLRepository) FindRuleByID(ctx context.Context, ruleID uint64) (Rule, error) {
	var row database.MailExtractorRuleRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", ruleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Rule{}, ErrRuleNotFound
		}
		return Rule{}, err
	}
	return mapRuleRow(row)
}

func (r *MySQLRepository) DeleteRule(ctx context.Context, ruleID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&database.UserMailExtractorTemplateRow{}, "rule_id = ?", ruleID).Error; err != nil {
			return err
		}
		result := tx.Delete(&database.MailExtractorRuleRow{}, "id = ?", ruleID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRuleNotFound
		}
		return nil
	})
}

func (r *MySQLRepository) ListEnabledTemplateIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	var rows []database.UserMailExtractorTemplateRow
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND enabled = ?", userID, true).
		Order("rule_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]uint64, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.RuleID)
	}
	return items, nil
}

func (r *MySQLRepository) SetTemplateEnabled(ctx context.Context, userID uint64, ruleID uint64, enabled bool) error {
	row := database.UserMailExtractorTemplateRow{
		UserID:  userID,
		RuleID:  ruleID,
		Enabled: enabled,
	}
	return r.db.WithContext(ctx).Save(&row).Error
}

func mapRuleRows(rows []database.MailExtractorRuleRow) ([]Rule, error) {
	items := make([]Rule, 0, len(rows))
	for _, row := range rows {
		item, err := mapRuleRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func mapRuleRow(row database.MailExtractorRuleRow) (Rule, error) {
	targetFields := make([]TargetField, 0)
	if len(row.TargetFieldsJSON) > 0 {
		if err := json.Unmarshal(row.TargetFieldsJSON, &targetFields); err != nil {
			return Rule{}, err
		}
	}

	mailboxIDs := make([]uint64, 0)
	if len(row.MailboxScopeJSON) > 0 {
		if err := json.Unmarshal(row.MailboxScopeJSON, &mailboxIDs); err != nil {
			return Rule{}, err
		}
	}

	domainIDs := make([]uint64, 0)
	if len(row.DomainScopeJSON) > 0 {
		if err := json.Unmarshal(row.DomainScopeJSON, &domainIDs); err != nil {
			return Rule{}, err
		}
	}

	return Rule{
		ID:                row.ID,
		OwnerUserID:       row.OwnerUserID,
		SourceType:        RuleSourceType(row.SourceType),
		TemplateKey:       row.TemplateKey,
		Name:              row.Name,
		Description:       row.Description,
		Label:             row.Label,
		Enabled:           row.Enabled,
		TargetFields:      targetFields,
		Pattern:           row.Pattern,
		Flags:             row.Flags,
		ResultMode:        ResultMode(row.ResultMode),
		CaptureGroupIndex: row.CaptureGroupIndex,
		MailboxIDs:        mailboxIDs,
		DomainIDs:         domainIDs,
		SenderContains:    row.SenderContains,
		SubjectContains:   row.SubjectContains,
		SortOrder:         row.SortOrder,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

func mapRuleToRow(rule Rule) (database.MailExtractorRuleRow, error) {
	targetFieldsJSON, err := json.Marshal(rule.TargetFields)
	if err != nil {
		return database.MailExtractorRuleRow{}, err
	}
	mailboxScopeJSON, err := json.Marshal(rule.MailboxIDs)
	if err != nil {
		return database.MailExtractorRuleRow{}, err
	}
	domainScopeJSON, err := json.Marshal(rule.DomainIDs)
	if err != nil {
		return database.MailExtractorRuleRow{}, err
	}
	return database.MailExtractorRuleRow{
		ID:                rule.ID,
		OwnerUserID:       rule.OwnerUserID,
		SourceType:        string(rule.SourceType),
		TemplateKey:       rule.TemplateKey,
		Name:              rule.Name,
		Description:       rule.Description,
		Label:             rule.Label,
		Enabled:           rule.Enabled,
		TargetFieldsJSON:  targetFieldsJSON,
		Pattern:           rule.Pattern,
		Flags:             rule.Flags,
		ResultMode:        string(rule.ResultMode),
		CaptureGroupIndex: rule.CaptureGroupIndex,
		MailboxScopeJSON:  mailboxScopeJSON,
		DomainScopeJSON:   domainScopeJSON,
		SenderContains:    rule.SenderContains,
		SubjectContains:   rule.SubjectContains,
		SortOrder:         rule.SortOrder,
	}, nil
}
