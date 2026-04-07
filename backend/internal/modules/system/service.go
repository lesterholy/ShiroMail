package system

import (
	"context"
	"sort"
	"strings"
)

type Service struct {
	configRepo ConfigRepository
	jobRepo    JobRepository
	auditRepo  AuditRepository
	mailTester MailDeliveryTester
}

func NewService(configRepo ConfigRepository, jobRepo JobRepository, auditRepo AuditRepository, options ...any) *Service {
	var mailTester MailDeliveryTester
	for _, option := range options {
		if current, ok := option.(MailDeliveryTester); ok {
			mailTester = current
		}
	}
	if mailTester == nil {
		mailTester = NewConfigMailDeliveryTester(configRepo)
	}
	return &Service{
		configRepo: configRepo,
		jobRepo:    jobRepo,
		auditRepo:  auditRepo,
		mailTester: mailTester,
	}
}

func (s *Service) ListConfigs(ctx context.Context) ([]ConfigEntry, error) {
	items, err := s.configRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	return mergeConfigEntries(items), nil
}

func (s *Service) UpsertConfig(ctx context.Context, actorID uint64, key string, value map[string]any) (ConfigEntry, error) {
	value = normalizeConfigValue(key, value)
	item, err := s.configRepo.Upsert(ctx, key, value, actorID)
	if err != nil {
		return ConfigEntry{}, err
	}
	_, _ = s.auditRepo.Create(ctx, actorID, "admin.config.upsert", "config", key, value)
	return normalizeConfigEntry(item), nil
}

func (s *Service) DeleteConfig(ctx context.Context, key string) error {
	return s.configRepo.Delete(ctx, key)
}

func (s *Service) ListSettingsSections(ctx context.Context) ([]SettingsSection, error) {
	items, err := s.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}

	index := make(map[string]ConfigEntry, len(items))
	for _, item := range items {
		index[item.Key] = item
	}

	sections := make([]SettingsSection, 0, len(settingsSectionDefinitions))
	for _, definition := range settingsSectionDefinitions {
		configKeys := definition.ConfigKeys
		if definition.Key == "oauth" {
			configKeys = buildOAuthSectionKeys(items)
		}

		sectionItems := make([]ConfigEntry, 0, len(configKeys))
		for _, key := range configKeys {
			item, ok := index[key]
			if !ok {
				item = ConfigEntry{Key: key, Value: defaultConfigValueForKey(key)}
			}
			sectionItems = append(sectionItems, normalizeConfigEntry(item))
		}

		sections = append(sections, SettingsSection{
			Key:         definition.Key,
			Title:       definition.Title,
			Description: definition.Description,
			Items:       sectionItems,
		})
	}

	return sections, nil
}

func buildOAuthSectionKeys(items []ConfigEntry) []string {
	keys := []string{ConfigKeyAuthOAuthDisplay}
	seen := map[string]struct{}{
		ConfigKeyAuthOAuthDisplay: {},
	}

	for _, item := range items {
		if !strings.HasPrefix(item.Key, "auth.oauth.providers.") {
			continue
		}
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		keys = append(keys, item.Key)
	}

	return keys
}

func (s *Service) PublicSiteSettings(ctx context.Context) (PublicSiteSettings, error) {
	return LoadPublicSiteSettings(ctx, s.configRepo)
}

func (s *Service) SendMailDeliveryTest(ctx context.Context, actorID uint64, to string) (MailDeliveryTestResult, error) {
	settings, err := LoadMailDeliverySettings(ctx, s.configRepo)
	if err != nil {
		return MailDeliveryTestResult{}, err
	}
	recipient := strings.TrimSpace(to)
	if recipient == "" {
		recipient = settings.FromAddress
	}
	if err := s.mailTester.SendTestMail(ctx, recipient); err != nil {
		return MailDeliveryTestResult{}, err
	}
	_, _ = s.auditRepo.Create(ctx, actorID, "admin.mail_delivery.test", "config", ConfigKeyMailDelivery, map[string]any{
		"recipient": recipient,
	})
	return MailDeliveryTestResult{
		Status:    "ok",
		Recipient: recipient,
	}, nil
}

func (s *Service) ListJobs(ctx context.Context) ([]JobRecord, error) {
	return s.jobRepo.List(ctx)
}

func (s *Service) ListAudit(ctx context.Context) ([]AuditLog, error) {
	return s.auditRepo.List(ctx)
}

func mergeConfigEntries(items []ConfigEntry) []ConfigEntry {
	index := make(map[string]ConfigEntry, len(items))
	for _, item := range defaultConfigEntries() {
		index[item.Key] = normalizeConfigEntry(item)
	}
	for _, item := range items {
		index[item.Key] = normalizeConfigEntry(item)
	}

	keys := make([]string, 0, len(index))
	for key := range index {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	output := make([]ConfigEntry, 0, len(keys))
	for _, key := range keys {
		output = append(output, index[key])
	}
	return output
}
