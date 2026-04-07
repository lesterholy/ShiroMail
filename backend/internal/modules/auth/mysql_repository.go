package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"shiro-email/backend/internal/database"
)

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) CreateUser(ctx context.Context, user User) (User, error) {
	if err := r.ensureUniqueUserFields(ctx, user.Username, user.Email); err != nil {
		return User{}, err
	}

	created := User{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := database.UserRow{
			Username:      user.Username,
			Email:         user.Email,
			PasswordHash:  user.PasswordHash,
			Status:        firstNonEmpty(user.Status, "active"),
			EmailVerified: user.EmailVerified,
		}
		if err := tx.Create(&row).Error; err != nil {
			return mapAuthWriteError(err)
		}

		roleIDs, err := r.lookupRoleIDs(ctx, tx, user.Roles)
		if err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			if err := tx.Create(&database.UserRoleRow{UserID: row.ID, RoleID: roleID}).Error; err != nil {
				return fmt.Errorf("create user role: %w", err)
			}
		}

		created = User{
			ID:            row.ID,
			Username:      row.Username,
			Email:         row.Email,
			PasswordHash:  row.PasswordHash,
			Status:        row.Status,
			EmailVerified: row.EmailVerified,
			Roles:         slices.Clone(user.Roles),
		}
		return nil
	})
	if err != nil {
		return User{}, err
	}
	return created, nil
}

func (r *MySQLRepository) FindUserByLogin(ctx context.Context, login string) (User, error) {
	var row database.UserRow
	result := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", login, login).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return User{}, result.Error
	}
	if result.RowsAffected == 0 {
		return User{}, ErrUserNotFound
	}

	return r.loadUser(ctx, row)
}

func (r *MySQLRepository) FindUserByEmail(ctx context.Context, email string) (User, error) {
	var row database.UserRow
	result := r.db.WithContext(ctx).
		Where("email = ?", email).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return User{}, result.Error
	}
	if result.RowsAffected == 0 {
		return User{}, ErrUserNotFound
	}

	return r.loadUser(ctx, row)
}

func (r *MySQLRepository) FindUserByID(ctx context.Context, id uint64) (User, error) {
	var row database.UserRow
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Limit(1).
		Find(&row)
	if result.Error != nil {
		return User{}, result.Error
	}
	if result.RowsAffected == 0 {
		return User{}, ErrUserNotFound
	}

	return r.loadUser(ctx, row)
}

func (r *MySQLRepository) ListUsers(ctx context.Context) ([]User, error) {
	var rows []database.UserRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]User, 0, len(rows))
	for _, row := range rows {
		item, err := r.loadUser(ctx, row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *MySQLRepository) UpdateUserRoles(ctx context.Context, id uint64, roles []string) (User, error) {
	normalizedRoles := uniqueRoleCodes(roles)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row database.UserRow
		result := tx.Where("id = ?", id).Limit(1).Find(&row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}

		roleIDs, err := r.lookupRoleIDs(ctx, tx, normalizedRoles)
		if err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.UserRoleRow{}).Error; err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			if err := tx.Create(&database.UserRoleRow{UserID: id, RoleID: roleID}).Error; err != nil {
				return fmt.Errorf("create user role: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return User{}, err
	}

	return r.FindUserByID(ctx, id)
}

func (r *MySQLRepository) UpdateUser(ctx context.Context, user User) (User, error) {
	if err := r.ensureUniqueUsername(ctx, user.Username, user.ID); err != nil {
		return User{}, err
	}
	if err := r.ensureUniqueEmail(ctx, user.Email, user.ID); err != nil {
		return User{}, err
	}

	result := r.db.WithContext(ctx).
		Model(&database.UserRow{}).
		Where("id = ?", user.ID).
		Updates(map[string]any{
			"username":       user.Username,
			"email":          user.Email,
			"status":         user.Status,
			"email_verified": user.EmailVerified,
		})
	if result.Error != nil {
		return User{}, mapAuthWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return User{}, ErrUserNotFound
	}
	return r.FindUserByID(ctx, user.ID)
}

func (r *MySQLRepository) UpdateUserPassword(ctx context.Context, id uint64, passwordHash string) error {
	result := r.db.WithContext(ctx).
		Model(&database.UserRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash": passwordHash,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MySQLRepository) UpdateUserVerification(ctx context.Context, id uint64, emailVerified bool, status string) error {
	result := r.db.WithContext(ctx).
		Model(&database.UserRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"email_verified": emailVerified,
			"status":         status,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MySQLRepository) UpdateUserEmail(ctx context.Context, id uint64, email string, emailVerified bool) error {
	if err := r.ensureUniqueEmail(ctx, email, id); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).
		Model(&database.UserRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"email":          email,
			"email_verified": emailVerified,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MySQLRepository) RefreshPendingRegistration(ctx context.Context, id uint64, username string, passwordHash string) (User, error) {
	if err := r.ensureUniqueUsername(ctx, username, id); err != nil {
		return User{}, err
	}

	result := r.db.WithContext(ctx).
		Model(&database.UserRow{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"username":       username,
			"password_hash":  passwordHash,
			"status":         "pending_verification",
			"email_verified": false,
		})
	if result.Error != nil {
		return User{}, mapAuthWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return User{}, ErrUserNotFound
	}
	return r.FindUserByID(ctx, id)
}

func (r *MySQLRepository) DeleteUser(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&database.UserRoleRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.AuthEmailVerificationRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.UserTOTPCredentialRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.AuthMFAChallengeRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.UserProfileRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.FeedbackRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("api_key_id IN (?)",
			tx.Model(&database.APIKeyRow{}).Select("id").Where("user_id = ?", id),
		).Delete(&database.APIKeyDomainBindingRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("api_key_id IN (?)",
			tx.Model(&database.APIKeyRow{}).Select("id").Where("user_id = ?", id),
		).Delete(&database.APIKeyScopeRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("api_key_id IN (?)",
			tx.Model(&database.APIKeyRow{}).Select("id").Where("user_id = ?", id),
		).Delete(&database.APIKeyResourcePolicyRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.APIKeyRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.WebhookRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.BillingProfileRow{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&database.BalanceEntryRow{}).Error; err != nil {
			return err
		}

		result := tx.Where("id = ?", id).Delete(&database.UserRow{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}
		return nil
	})
}

func (r *MySQLRepository) loadUser(ctx context.Context, row database.UserRow) (User, error) {
	roles, err := r.loadRoleCodes(ctx, r.db, row.ID)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:            row.ID,
		Username:      row.Username,
		Email:         row.Email,
		PasswordHash:  row.PasswordHash,
		Status:        row.Status,
		EmailVerified: row.EmailVerified,
		Roles:         roles,
	}, nil
}

func (r *MySQLRepository) SaveEmailVerification(ctx context.Context, record EmailVerificationRecord) (EmailVerificationRecord, error) {
	row := database.AuthEmailVerificationRow{
		UserID:     record.UserID,
		Email:      record.Email,
		Purpose:    record.Purpose,
		TicketHash: record.TicketHash,
		CodeHash:   record.CodeHash,
		ExpiresAt:  record.ExpiresAt,
		ConsumedAt: record.ConsumedAt,
		LastSentAt: record.LastSentAt,
		Attempts:   record.Attempts,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return EmailVerificationRecord{}, err
	}
	record.ID = row.ID
	return record, nil
}

func (r *MySQLRepository) FindEmailVerificationByTicketHash(ctx context.Context, ticketHash string) (EmailVerificationRecord, error) {
	var row database.AuthEmailVerificationRow
	result := r.db.WithContext(ctx).Where("ticket_hash = ?", ticketHash).Limit(1).Find(&row)
	if result.Error != nil {
		return EmailVerificationRecord{}, result.Error
	}
	if result.RowsAffected == 0 {
		return EmailVerificationRecord{}, ErrUserNotFound
	}
	return mapVerificationRow(row), nil
}

func (r *MySQLRepository) ConsumeEmailVerification(ctx context.Context, id uint64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&database.AuthEmailVerificationRow{}).
		Where("id = ?", id).
		Update("consumed_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MySQLRepository) IncrementEmailVerificationAttempts(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).
		Model(&database.AuthEmailVerificationRow{}).
		Where("id = ?", id).
		Update("attempts", gorm.Expr("attempts + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MySQLRepository) UpsertTOTPCredential(ctx context.Context, record TOTPCredential) error {
	row := database.UserTOTPCredentialRow{
		UserID:           record.UserID,
		SecretCiphertext: record.SecretCiphertext,
		Enabled:          record.Enabled,
		VerifiedAt:       record.VerifiedAt,
		LastUsedAt:       record.LastUsedAt,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"secret_ciphertext": row.SecretCiphertext,
				"enabled":           row.Enabled,
				"verified_at":       row.VerifiedAt,
				"last_used_at":      row.LastUsedAt,
			}),
		}).
		Create(&row).Error
}

func (r *MySQLRepository) FindTOTPCredentialByUserID(ctx context.Context, userID uint64) (TOTPCredential, error) {
	var row database.UserTOTPCredentialRow
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(1).Find(&row)
	if result.Error != nil {
		return TOTPCredential{}, result.Error
	}
	if result.RowsAffected == 0 {
		return TOTPCredential{}, ErrUserNotFound
	}
	return TOTPCredential{
		UserID:           row.UserID,
		SecretCiphertext: row.SecretCiphertext,
		Enabled:          row.Enabled,
		VerifiedAt:       row.VerifiedAt,
		LastUsedAt:       row.LastUsedAt,
	}, nil
}

func (r *MySQLRepository) SaveMFAChallenge(ctx context.Context, record MFAChallengeRecord) (MFAChallengeRecord, error) {
	row := database.AuthMFAChallengeRow{
		UserID:     record.UserID,
		TicketHash: record.TicketHash,
		Purpose:    record.Purpose,
		ExpiresAt:  record.ExpiresAt,
		ConsumedAt: record.ConsumedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return MFAChallengeRecord{}, err
	}
	record.ID = row.ID
	return record, nil
}

func (r *MySQLRepository) FindMFAChallengeByTicketHash(ctx context.Context, ticketHash string) (MFAChallengeRecord, error) {
	var row database.AuthMFAChallengeRow
	result := r.db.WithContext(ctx).Where("ticket_hash = ?", ticketHash).Limit(1).Find(&row)
	if result.Error != nil {
		return MFAChallengeRecord{}, result.Error
	}
	if result.RowsAffected == 0 {
		return MFAChallengeRecord{}, ErrUserNotFound
	}
	return MFAChallengeRecord{
		ID:         row.ID,
		UserID:     row.UserID,
		TicketHash: row.TicketHash,
		Purpose:    row.Purpose,
		ExpiresAt:  row.ExpiresAt,
		ConsumedAt: row.ConsumedAt,
	}, nil
}

func (r *MySQLRepository) ConsumeMFAChallenge(ctx context.Context, id uint64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&database.AuthMFAChallengeRow{}).
		Where("id = ?", id).
		Update("consumed_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *MySQLRepository) GetProfileSettings(ctx context.Context, userID uint64) (ProfileSettings, error) {
	var row database.UserProfileRow
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(1).Find(&row)
	if result.Error != nil {
		return ProfileSettings{}, result.Error
	}
	if result.RowsAffected == 0 {
		return ProfileSettings{}, ErrUserNotFound
	}
	return ProfileSettings{
		UserID:             row.UserID,
		DisplayName:        row.DisplayName,
		Locale:             row.Locale,
		Timezone:           row.Timezone,
		AutoRefreshSeconds: row.AutoRefreshSeconds,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

func (r *MySQLRepository) UpsertProfileSettings(ctx context.Context, item ProfileSettings) (ProfileSettings, error) {
	row := database.UserProfileRow{
		UserID:             item.UserID,
		DisplayName:        item.DisplayName,
		Locale:             item.Locale,
		Timezone:           item.Timezone,
		AutoRefreshSeconds: item.AutoRefreshSeconds,
	}
	if err := r.db.WithContext(ctx).Where("user_id = ?", item.UserID).Assign(row).FirstOrCreate(&row).Error; err != nil {
		return ProfileSettings{}, err
	}
	return r.GetProfileSettings(ctx, item.UserID)
}

func (r *MySQLRepository) ensureUniqueUserFields(ctx context.Context, username string, email string) error {
	if err := r.ensureUniqueUsername(ctx, username, 0); err != nil {
		return err
	}
	return r.ensureUniqueEmail(ctx, email, 0)
}

func (r *MySQLRepository) ensureUniqueUsername(ctx context.Context, username string, excludeUserID uint64) error {
	if strings.TrimSpace(username) == "" {
		return nil
	}

	var count int64
	query := r.db.WithContext(ctx).Model(&database.UserRow{}).Where("username = ?", username)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("username already exists")
	}
	return nil
}

func (r *MySQLRepository) ensureUniqueEmail(ctx context.Context, email string, excludeUserID uint64) error {
	if strings.TrimSpace(email) == "" {
		return nil
	}

	var count int64
	query := r.db.WithContext(ctx).Model(&database.UserRow{}).Where("email = ?", email)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("email already exists")
	}
	return nil
}

func (r *MySQLRepository) lookupRoleIDs(ctx context.Context, db *gorm.DB, codes []string) ([]uint64, error) {
	if len(codes) == 0 {
		return nil, nil
	}

	var rows []database.RoleRow
	if err := db.WithContext(ctx).Where("code IN ?", codes).Find(&rows).Error; err != nil {
		return nil, err
	}

	byCode := make(map[string]uint64, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row.ID
	}

	ids := make([]uint64, 0, len(codes))
	for _, code := range codes {
		roleID, ok := byCode[code]
		if !ok {
			return nil, fmt.Errorf("role not found: %s", code)
		}
		ids = append(ids, roleID)
	}
	return ids, nil
}

func (r *MySQLRepository) loadRoleCodes(ctx context.Context, db *gorm.DB, userID uint64) ([]string, error) {
	var rows []database.RoleRow
	if err := db.WithContext(ctx).
		Table("roles").
		Select("roles.id, roles.code, roles.name").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Order("roles.code ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, row.Code)
	}
	return codes, nil
}

func mapAuthWriteError(err error) error {
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1062 {
		return err
	}

	message := strings.ToLower(mysqlErr.Message)
	switch {
	case strings.Contains(message, "username"):
		return errors.New("username already exists")
	case strings.Contains(message, "email"):
		return errors.New("email already exists")
	default:
		return errors.New("user already exists")
	}
}

func uniqueRoleCodes(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(codes))
	items := make([]string, 0, len(codes))
	for _, code := range codes {
		normalized := strings.TrimSpace(code)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	slices.Sort(items)
	return items
}

func mapVerificationRow(row database.AuthEmailVerificationRow) EmailVerificationRecord {
	return EmailVerificationRecord{
		ID:         row.ID,
		UserID:     row.UserID,
		Email:      row.Email,
		Purpose:    row.Purpose,
		TicketHash: row.TicketHash,
		CodeHash:   row.CodeHash,
		ExpiresAt:  row.ExpiresAt,
		ConsumedAt: row.ConsumedAt,
		LastSentAt: row.LastSentAt,
		Attempts:   row.Attempts,
	}
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
