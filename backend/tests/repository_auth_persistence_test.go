package tests

import (
	"context"
	"testing"
	"time"

	"shiro-email/backend/internal/database"
	"shiro-email/backend/internal/modules/auth"
	"shiro-email/backend/internal/modules/portal"
)

func TestMySQLAuthRepositoryCreatesAndFindsUser(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("expected schema bootstrap success, got %v", err)
	}
	mustSeedRoles(t, db, "user", "admin")

	repo := auth.NewMySQLRepository(db)
	created, err := repo.CreateUser(ctx, auth.User{
		Username:     "persisted-user",
		Email:        "persisted-user@example.com",
		PasswordHash: "hash",
		Roles:        []string{"user"},
	})
	if err != nil {
		t.Fatalf("expected create success, got %v", err)
	}

	found, err := repo.FindUserByLogin(ctx, "persisted-user")
	if err != nil {
		t.Fatalf("expected find success, got %v", err)
	}
	if found.ID != created.ID || len(found.Roles) != 1 || found.Roles[0] != "user" {
		t.Fatalf("unexpected found user: %+v", found)
	}
}

func TestMySQLAuthRepositoryDeletesUserWithAPIKeyDomainBindings(t *testing.T) {
	db := mustOpenTestMySQL(t)
	ctx := context.Background()

	mustResetPersistenceTables(t, db)
	if err := database.EnsureSchema(ctx, db); err != nil {
		t.Fatalf("expected schema bootstrap success, got %v", err)
	}
	mustSeedRoles(t, db, "user", "admin")

	userID := mustSeedUser(t, db, "delete-api-user", "delete-api-user@example.com", []string{"user"})
	authRepo := auth.NewMySQLRepository(db)
	portalRepo := portal.NewMySQLRepository(db)

	zoneID := uint64(99)
	_, err := portalRepo.CreateAPIKey(ctx, portal.APIKey{
		UserID:     userID,
		Name:       "bound-key",
		KeyPrefix:  "sk_live",
		KeyPreview: "sk_live_preview",
		SecretHash: "secret-hash",
		Status:     "active",
		Scopes:     []string{"mailboxes.read"},
		DomainBindings: []portal.APIKeyDomainBinding{
			{
				ZoneID:      &zoneID,
				AccessLevel: "read",
			},
		},
	})
	if err != nil {
		t.Fatalf("expected api key create success, got %v", err)
	}

	if err := authRepo.DeleteUser(ctx, userID); err != nil {
		t.Fatalf("expected delete user success, got %v", err)
	}

	if _, err := authRepo.FindUserByID(ctx, userID); err == nil {
		t.Fatal("expected deleted user lookup to fail")
	}
}

func TestRedisRefreshStoreSaveFindAndRevoke(t *testing.T) {
	client := mustOpenTestRedis(t)
	ctx := context.Background()

	mustFlushRedis(t, client)

	store := auth.NewRedisRefreshStore(client)
	record := auth.RefreshTokenRecord{
		UserID:    7,
		TokenHash: "token-hash",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := store.SaveRefreshToken(ctx, record); err != nil {
		t.Fatalf("expected save success, got %v", err)
	}
	if _, err := store.FindRefreshToken(ctx, record.TokenHash); err != nil {
		t.Fatalf("expected find success, got %v", err)
	}
	if err := store.RevokeRefreshToken(ctx, record.TokenHash); err != nil {
		t.Fatalf("expected revoke success, got %v", err)
	}
	if _, err := store.FindRefreshToken(ctx, record.TokenHash); err == nil {
		t.Fatal("expected revoked token lookup to fail")
	}
}
