package portal

import (
	"context"
	"strings"
	"time"

	"shiro-email/backend/internal/database"
	"shiro-email/backend/internal/modules/auth"
)

func EnsureDemoData(ctx context.Context, repo Repository, user auth.User) error {
	defaults := defaultProfileSettings(user)

	profile, err := repo.GetProfileSettings(ctx, user.ID)
	if err != nil {
		if _, err := repo.UpsertProfileSettings(ctx, defaults); err != nil {
			return err
		}
	} else if shouldRepairSeedProfile(profile) {
		profile.DisplayName = defaults.DisplayName
		profile.Locale = defaults.Locale
		profile.Timezone = defaults.Timezone
		if profile.AutoRefreshSeconds <= 0 {
			profile.AutoRefreshSeconds = defaults.AutoRefreshSeconds
		}
		if _, err := repo.UpsertProfileSettings(ctx, profile); err != nil {
			return err
		}
	}

	switch typed := repo.(type) {
	case *MySQLRepository:
		return typed.seedMySQL(ctx, user.ID)
	case *MemoryRepository:
		return typed.seedMemory(user.ID)
	default:
		return nil
	}
}

func defaultProfileSettings(user auth.User) ProfileSettings {
	displayName := strings.TrimSpace(user.Username)
	if displayName == "" {
		displayName = strings.TrimSpace(user.Email)
	}
	if at := strings.Index(displayName, "@"); at > 0 {
		displayName = strings.TrimSpace(displayName[:at])
	}
	if displayName == "" {
		displayName = "user"
	}

	return ProfileSettings{
		UserID:             user.ID,
		DisplayName:        displayName,
		Locale:             "zh-CN",
		Timezone:           "Asia/Shanghai",
		AutoRefreshSeconds: 30,
	}
}

func shouldRepairSeedProfile(profile ProfileSettings) bool {
	displayName := strings.TrimSpace(profile.DisplayName)
	return displayName == "" || displayName == "GALA Workspace"
}

func (r *MemoryRepository) seedMemory(userID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.billing[userID]; !ok {
		r.billing[userID] = BillingProfile{
			UserID:            userID,
			PlanCode:          "free",
			PlanName:          "Free",
			Status:            "active",
			MailboxQuota:      3,
			DomainQuota:       1,
			DailyRequestLimit: 1000,
			RenewalAt:         time.Now(),
		}
	}

	return nil
}

func (r *MySQLRepository) seedMySQL(ctx context.Context, userID uint64) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).FirstOrCreate(&database.BillingProfileRow{
		UserID:            userID,
		PlanCode:          "free",
		PlanName:          "Free",
		Status:            "active",
		MailboxQuota:      3,
		DomainQuota:       1,
		DailyRequestLimit: 1000,
		RenewalAt:         time.Now(),
	}).Error; err != nil {
		return err
	}

	return nil
}
