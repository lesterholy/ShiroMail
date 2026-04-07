package tests

import (
	"context"
	"testing"

	"shiro-email/backend/internal/database"
	"shiro-email/backend/internal/modules/domain"
	"shiro-email/backend/internal/modules/portal"
	"shiro-email/backend/internal/modules/system"
)

func TestMySQLSystemRepositoriesPersistConfigJobAndAudit(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)
	userID := mustSeedUser(t, db, "admin-user", "admin-user@example.com", []string{"admin"})

	configRepo := system.NewMySQLConfigRepository(db)
	jobRepo := system.NewMySQLJobRepository(db)
	auditRepo := system.NewMySQLAuditRepository(db)

	if _, err := configRepo.Upsert(ctx, "platform", map[string]any{"brand": "Shiro Email"}, userID); err != nil {
		t.Fatalf("expected config upsert success, got %v", err)
	}
	if _, err := jobRepo.Create(ctx, "sync_messages", "failed", "timeout"); err != nil {
		t.Fatalf("expected job create success, got %v", err)
	}
	if _, err := auditRepo.Create(ctx, userID, "admin.config.upsert", "config", "platform", map[string]any{"brand": "Shiro Email"}); err != nil {
		t.Fatalf("expected audit create success, got %v", err)
	}
	if jobRepo.CountFailed(ctx) != 1 {
		t.Fatalf("expected one failed job")
	}
}

func TestDomainPlatformSchemaPersistsProviderAccountsAndAPIKeyPolicies(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)

	userID := mustSeedUser(t, db, "domain-owner", "domain-owner@example.com", []string{"user"})

	providerResult := db.WithContext(ctx).Exec(
		`INSERT INTO provider_accounts (provider, owner_type, owner_user_id, display_name, status)
		 VALUES (?, ?, ?, ?, ?)`,
		"cloudflare",
		"user",
		userID,
		"My Cloudflare",
		"healthy",
	)
	if providerResult.Error != nil {
		t.Fatalf("expected provider account create success, got %v", providerResult.Error)
	}

	apiKeyResult := db.WithContext(ctx).Exec(
		`INSERT INTO user_api_keys (user_id, name, key_prefix, key_preview, secret_hash, status)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID,
		"ci-key",
		"sk_live",
		"sk_live_xxx",
		"hash",
		"active",
	)
	if apiKeyResult.Error != nil {
		t.Fatalf("expected api key create success, got %v", apiKeyResult.Error)
	}

	var apiKeyRow struct {
		ID uint64
	}
	if err := db.WithContext(ctx).
		Raw("SELECT id FROM user_api_keys WHERE user_id = ? AND name = ?", userID, "ci-key").
		Scan(&apiKeyRow).Error; err != nil {
		t.Fatalf("expected api key lookup success, got %v", err)
	}

	policyResult := db.WithContext(ctx).Exec(
		`INSERT INTO api_key_resource_policies (
			api_key_id,
			domain_access_mode,
			allow_platform_public_domains,
			allow_user_published_domains,
			allow_owned_private_domains,
			allow_provider_mutation,
			allow_protected_record_write
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		apiKeyRow.ID,
		"mixed",
		true,
		true,
		true,
		false,
		false,
	)
	if policyResult.Error != nil {
		t.Fatalf("expected api key policy create success, got %v", policyResult.Error)
	}
}

func TestMySQLPortalRepositoryCreatesAPIKeyWithPolicy(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)

	userID := mustSeedUser(t, db, "api-owner", "api-owner@example.com", []string{"user"})
	repo := portal.NewMySQLRepository(db)

	zoneID := uint64(42)
	created, err := repo.CreateAPIKey(ctx, portal.APIKey{
		UserID:     userID,
		Name:       "worker",
		KeyPrefix:  "sk_live",
		KeyPreview: "sk_live_preview",
		SecretHash: "secret-hash",
		Status:     "active",
		Scopes:     []string{"mailboxes.read", "domains.verify"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:           "private_only",
			AllowOwnedPrivateDomains:   true,
			AllowPlatformPublicDomains: false,
		},
		DomainBindings: []portal.APIKeyDomainBinding{
			{
				ZoneID:      &zoneID,
				AccessLevel: "verify",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	if len(created.Scopes) != 2 || created.ResourcePolicy.DomainAccessMode != "private_only" {
		t.Fatalf("unexpected created api key: %+v", created)
	}

	items, err := repo.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		t.Fatalf("expected list success, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one api key, got %d", len(items))
	}
	if len(items[0].Scopes) != 2 {
		t.Fatalf("expected persisted scopes, got %+v", items[0].Scopes)
	}
	if items[0].ResourcePolicy.DomainAccessMode != "private_only" {
		t.Fatalf("expected persisted resource policy, got %+v", items[0].ResourcePolicy)
	}
	if len(items[0].DomainBindings) != 1 || items[0].DomainBindings[0].AccessLevel != "verify" {
		t.Fatalf("expected persisted domain bindings, got %+v", items[0].DomainBindings)
	}
}

func TestSystemServiceListsDefaultDomainPublicationPolicy(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)

	service := system.NewService(
		system.NewMySQLConfigRepository(db),
		system.NewMySQLJobRepository(db),
		system.NewMySQLAuditRepository(db),
	)

	items, err := service.ListConfigs(ctx)
	if err != nil {
		t.Fatalf("expected config list success, got %v", err)
	}

	for _, item := range items {
		if item.Key != system.ConfigKeyDomainPublicPoolPolicy {
			continue
		}

		requiresReview, ok := item.Value["requiresReview"].(bool)
		if !ok || !requiresReview {
			t.Fatalf("expected default publication review policy, got %+v", item.Value)
		}
		return
	}

	t.Fatalf("expected %s config to be present", system.ConfigKeyDomainPublicPoolPolicy)
}

func TestMySQLDomainRepositoryCreatesProviderAccountAndBindsRootDomain(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)

	repo := domain.NewMySQLRepository(db)
	provider, err := repo.CreateProviderAccount(ctx, domain.ProviderAccount{
		Provider:     "cloudflare",
		OwnerType:    "platform",
		DisplayName:  "Ops Cloudflare",
		AuthType:     "api_token",
		Status:       "healthy",
		Capabilities: []string{"zones.read", "dns.write"},
	})
	if err != nil {
		t.Fatalf("expected provider create success, got %v", err)
	}

	created, err := repo.Upsert(ctx, domain.Domain{
		Domain:            "corpzone.test",
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		HealthStatus:      "healthy",
		ProviderAccountID: &provider.ID,
	})
	if err != nil {
		t.Fatalf("expected domain upsert success, got %v", err)
	}
	if created.ProviderAccountID == nil || *created.ProviderAccountID != provider.ID {
		t.Fatalf("expected provider binding on created domain, got %+v", created)
	}

	items, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("expected domain list success, got %v", err)
	}
	if len(items) != 1 || items[0].ProviderDisplayName != "Ops Cloudflare" {
		t.Fatalf("expected bound provider in list, got %+v", items)
	}
}

func TestMySQLDomainRepositoryReusesExistingZoneWhenPreviewingProviderChangeSet(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	mustEnsureSchema(t, db)

	repo := domain.NewMySQLRepository(db)
	provider, err := repo.CreateProviderAccount(ctx, domain.ProviderAccount{
		Provider:     "cloudflare",
		OwnerType:    "platform",
		DisplayName:  "Ops Cloudflare",
		AuthType:     "api_token",
		Status:       "healthy",
		Capabilities: []string{"zones.read", "dns.write"},
	})
	if err != nil {
		t.Fatalf("expected provider create success, got %v", err)
	}

	if _, err := repo.Upsert(ctx, domain.Domain{
		Domain:            "galiais.online",
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		HealthStatus:      "healthy",
		ProviderAccountID: &provider.ID,
	}); err != nil {
		t.Fatalf("expected domain upsert success, got %v", err)
	}

	changeSet, err := repo.SaveDNSChangeSet(ctx, domain.DNSChangeSet{
		ProviderAccountID: provider.ID,
		ProviderZoneID:    "f367ab93421cca9e55a29f743917252f",
		ZoneName:          "galiais.online",
		Status:            "previewed",
		Provider:          "cloudflare",
		Summary:           "1 create",
		Operations: []domain.DNSChangeOperation{
			{
				Operation:  "create",
				RecordType: "TXT",
				RecordName: "_shiro-verification.galiais.online",
				After: &domain.ProviderRecord{
					Type:  "TXT",
					Name:  "_shiro-verification.galiais.online",
					Value: "shiro-ownership=galiais.online",
					TTL:   300,
				},
				Status: "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected change set save success, got %v", err)
	}
	if changeSet.ZoneID == nil {
		t.Fatal("expected saved change set to reference existing zone")
	}

	var zone database.DNSZoneRow
	if err := db.WithContext(ctx).Where("zone_name = ?", "galiais.online").First(&zone).Error; err != nil {
		t.Fatalf("expected zone lookup success, got %v", err)
	}
	if zone.ProviderZoneID != "f367ab93421cca9e55a29f743917252f" {
		t.Fatalf("expected provider zone id to be updated, got %q", zone.ProviderZoneID)
	}
	if zone.ProviderAccountID == nil || *zone.ProviderAccountID != provider.ID {
		t.Fatalf("expected zone provider binding to remain attached, got %+v", zone)
	}
}
