package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

type User struct {
	ID            uint64
	Username      string
	Email         string
	PasswordHash  string
	Status        string
	EmailVerified bool
	Roles         []string
}

type RefreshTokenRecord struct {
	UserID    uint64
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type EmailVerificationRecord struct {
	ID         uint64
	UserID     uint64
	Email      string
	Purpose    string
	TicketHash string
	CodeHash   string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	LastSentAt time.Time
	Attempts   int
}

type TOTPCredential struct {
	UserID           uint64
	SecretCiphertext string
	Enabled          bool
	VerifiedAt       *time.Time
	LastUsedAt       *time.Time
}

type MFAChallengeRecord struct {
	ID         uint64
	UserID     uint64
	TicketHash string
	Purpose    string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

type ProfileSettings struct {
	UserID             uint64
	DisplayName        string
	Locale             string
	Timezone           string
	AutoRefreshSeconds int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type UserStore interface {
	CreateUser(ctx context.Context, user User) (User, error)
	FindUserByLogin(ctx context.Context, login string) (User, error)
	FindUserByEmail(ctx context.Context, email string) (User, error)
	FindUserByID(ctx context.Context, id uint64) (User, error)
	ListUsers(ctx context.Context) ([]User, error)
	UpdateUserRoles(ctx context.Context, id uint64, roles []string) (User, error)
	UpdateUser(ctx context.Context, user User) (User, error)
	UpdateUserPassword(ctx context.Context, id uint64, passwordHash string) error
	UpdateUserVerification(ctx context.Context, id uint64, emailVerified bool, status string) error
	UpdateUserEmail(ctx context.Context, id uint64, email string, emailVerified bool) error
	RefreshPendingRegistration(ctx context.Context, id uint64, username string, passwordHash string) (User, error)
	DeleteUser(ctx context.Context, id uint64) error
}

type RefreshTokenStore interface {
	SaveRefreshToken(ctx context.Context, record RefreshTokenRecord) error
	FindRefreshToken(ctx context.Context, tokenHash string) (RefreshTokenRecord, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeUserRefreshTokens(ctx context.Context, userID uint64) error
}

type EmailVerificationStore interface {
	SaveEmailVerification(ctx context.Context, record EmailVerificationRecord) (EmailVerificationRecord, error)
	FindEmailVerificationByTicketHash(ctx context.Context, ticketHash string) (EmailVerificationRecord, error)
	ConsumeEmailVerification(ctx context.Context, id uint64) error
	IncrementEmailVerificationAttempts(ctx context.Context, id uint64) error
}

type TOTPStore interface {
	UpsertTOTPCredential(ctx context.Context, record TOTPCredential) error
	FindTOTPCredentialByUserID(ctx context.Context, userID uint64) (TOTPCredential, error)
}

type MFAChallengeStore interface {
	SaveMFAChallenge(ctx context.Context, record MFAChallengeRecord) (MFAChallengeRecord, error)
	FindMFAChallengeByTicketHash(ctx context.Context, ticketHash string) (MFAChallengeRecord, error)
	ConsumeMFAChallenge(ctx context.Context, id uint64) error
}

type ProfileSettingsStore interface {
	GetProfileSettings(ctx context.Context, userID uint64) (ProfileSettings, error)
	UpsertProfileSettings(ctx context.Context, item ProfileSettings) (ProfileSettings, error)
}

type Repository interface {
	UserStore
	RefreshTokenStore
	EmailVerificationStore
	TOTPStore
	MFAChallengeStore
	ProfileSettingsStore
}

type PersistentRepository struct {
	userStore         UserStore
	refreshStore      RefreshTokenStore
	verificationStore EmailVerificationStore
	totpStore         TOTPStore
	mfaStore          MFAChallengeStore
	profileStore      ProfileSettingsStore
}

type MemoryRepository struct {
	mu            sync.RWMutex
	nextUserID    uint64
	usersByID     map[uint64]User
	usersByName   map[string]uint64
	usersByEmail  map[string]uint64
	refreshByHash map[string]RefreshTokenRecord
	verifications map[string]EmailVerificationRecord
	totpByUserID  map[uint64]TOTPCredential
	mfaByHash     map[string]MFAChallengeRecord
	nextMFAID     uint64
	profiles      map[uint64]ProfileSettings
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nextUserID:    1,
		usersByID:     map[uint64]User{},
		usersByName:   map[string]uint64{},
		usersByEmail:  map[string]uint64{},
		refreshByHash: map[string]RefreshTokenRecord{},
		verifications: map[string]EmailVerificationRecord{},
		totpByUserID:  map[uint64]TOTPCredential{},
		mfaByHash:     map[string]MFAChallengeRecord{},
		nextMFAID:     1,
		profiles:      map[uint64]ProfileSettings{},
	}
}

func NewPersistentRepository(userStore UserStore, refreshStore RefreshTokenStore) *PersistentRepository {
	verificationStore, _ := userStore.(EmailVerificationStore)
	totpStore, _ := userStore.(TOTPStore)
	mfaStore, _ := userStore.(MFAChallengeStore)
	profileStore, _ := userStore.(ProfileSettingsStore)
	return &PersistentRepository{
		userStore:         userStore,
		refreshStore:      refreshStore,
		verificationStore: verificationStore,
		totpStore:         totpStore,
		mfaStore:          mfaStore,
		profileStore:      profileStore,
	}
}

func (r *MemoryRepository) CreateUser(_ context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.usersByName[user.Username]; exists {
		return User{}, errors.New("username already exists")
	}
	if _, exists := r.usersByEmail[user.Email]; exists {
		return User{}, errors.New("email already exists")
	}
	user.ID = r.nextUserID
	r.nextUserID++
	if user.Status == "" {
		user.Status = "active"
	}
	r.usersByID[user.ID] = user
	r.usersByName[user.Username] = user.ID
	r.usersByEmail[user.Email] = user.ID
	return user, nil
}

func (r *MemoryRepository) FindUserByLogin(_ context.Context, login string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id, ok := r.usersByName[login]; ok {
		return r.usersByID[id], nil
	}
	if id, ok := r.usersByEmail[login]; ok {
		return r.usersByID[id], nil
	}
	return User{}, ErrUserNotFound
}

func (r *MemoryRepository) FindUserByEmail(_ context.Context, email string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id, ok := r.usersByEmail[email]; ok {
		return r.usersByID[id], nil
	}
	return User{}, ErrUserNotFound
}

func (r *MemoryRepository) FindUserByID(_ context.Context, id uint64) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.usersByID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (r *MemoryRepository) ListUsers(_ context.Context) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]User, 0, len(r.usersByID))
	for _, user := range r.usersByID {
		items = append(items, user)
	}
	return items, nil
}

func (r *MemoryRepository) UpdateUserRoles(_ context.Context, id uint64, roles []string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.usersByID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	item.Roles = append([]string(nil), roles...)
	r.usersByID[id] = item
	return item, nil
}

func (r *MemoryRepository) UpdateUser(_ context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.usersByID[user.ID]
	if !ok {
		return User{}, ErrUserNotFound
	}
	if existingID, exists := r.usersByName[user.Username]; exists && existingID != user.ID {
		return User{}, errors.New("username already exists")
	}
	if existingID, exists := r.usersByEmail[user.Email]; exists && existingID != user.ID {
		return User{}, errors.New("email already exists")
	}

	if item.Username != user.Username {
		delete(r.usersByName, item.Username)
		r.usersByName[user.Username] = user.ID
	}
	if item.Email != user.Email {
		delete(r.usersByEmail, item.Email)
		r.usersByEmail[user.Email] = user.ID
	}

	item.Username = user.Username
	item.Email = user.Email
	item.Status = user.Status
	item.EmailVerified = user.EmailVerified
	r.usersByID[user.ID] = item
	return item, nil
}

func (r *MemoryRepository) UpdateUserPassword(_ context.Context, id uint64, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.usersByID[id]
	if !ok {
		return ErrUserNotFound
	}
	item.PasswordHash = passwordHash
	r.usersByID[id] = item
	return nil
}

func (r *MemoryRepository) UpdateUserVerification(_ context.Context, id uint64, emailVerified bool, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.usersByID[id]
	if !ok {
		return ErrUserNotFound
	}
	item.EmailVerified = emailVerified
	item.Status = status
	r.usersByID[id] = item
	return nil
}

func (r *MemoryRepository) UpdateUserEmail(_ context.Context, id uint64, email string, emailVerified bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, exists := r.usersByEmail[email]; exists && existingID != id {
		return errors.New("email already exists")
	}

	item, ok := r.usersByID[id]
	if !ok {
		return ErrUserNotFound
	}
	delete(r.usersByEmail, item.Email)
	item.Email = email
	item.EmailVerified = emailVerified
	r.usersByID[id] = item
	r.usersByEmail[email] = id
	return nil
}

func (r *MemoryRepository) RefreshPendingRegistration(_ context.Context, id uint64, username string, passwordHash string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.usersByID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	if existingID, exists := r.usersByName[username]; exists && existingID != id {
		return User{}, errors.New("username already exists")
	}

	if item.Username != username {
		delete(r.usersByName, item.Username)
		item.Username = username
		r.usersByName[username] = id
	}
	item.PasswordHash = passwordHash
	item.Status = "pending_verification"
	item.EmailVerified = false
	r.usersByID[id] = item
	return item, nil
}

func (r *MemoryRepository) DeleteUser(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.usersByID[id]
	if !ok {
		return ErrUserNotFound
	}
	delete(r.usersByID, id)
	delete(r.usersByName, item.Username)
	delete(r.usersByEmail, item.Email)
	delete(r.totpByUserID, id)
	delete(r.profiles, id)

	for key, record := range r.refreshByHash {
		if record.UserID == id {
			delete(r.refreshByHash, key)
		}
	}
	for key, record := range r.verifications {
		if record.UserID == id {
			delete(r.verifications, key)
		}
	}
	for key, record := range r.mfaByHash {
		if record.UserID == id {
			delete(r.mfaByHash, key)
		}
	}
	return nil
}

func (r *MemoryRepository) SaveRefreshToken(_ context.Context, record RefreshTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshByHash[record.TokenHash] = record
	return nil
}

func (r *MemoryRepository) FindRefreshToken(_ context.Context, tokenHash string) (RefreshTokenRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.refreshByHash[tokenHash]
	if !ok {
		return RefreshTokenRecord{}, ErrRefreshTokenNotFound
	}
	return record, nil
}

func (r *MemoryRepository) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.refreshByHash[tokenHash]
	if !ok {
		return ErrRefreshTokenNotFound
	}
	now := time.Now()
	record.RevokedAt = &now
	r.refreshByHash[tokenHash] = record
	return nil
}

func (r *MemoryRepository) RevokeUserRefreshTokens(_ context.Context, userID uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for tokenHash, record := range r.refreshByHash {
		if record.UserID != userID {
			continue
		}
		record.RevokedAt = &now
		r.refreshByHash[tokenHash] = record
	}
	return nil
}

func (r *MemoryRepository) SaveEmailVerification(_ context.Context, record EmailVerificationRecord) (EmailVerificationRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if record.ID == 0 {
		record.ID = uint64(len(r.verifications) + 1)
	}
	r.verifications[record.TicketHash] = record
	return record, nil
}

func (r *MemoryRepository) FindEmailVerificationByTicketHash(_ context.Context, ticketHash string) (EmailVerificationRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.verifications[ticketHash]
	if !ok {
		return EmailVerificationRecord{}, ErrUserNotFound
	}
	return record, nil
}

func (r *MemoryRepository) ConsumeEmailVerification(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.verifications {
		if item.ID != id {
			continue
		}
		now := time.Now()
		item.ConsumedAt = &now
		r.verifications[key] = item
		return nil
	}
	return ErrUserNotFound
}

func (r *MemoryRepository) IncrementEmailVerificationAttempts(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.verifications {
		if item.ID != id {
			continue
		}
		item.Attempts++
		r.verifications[key] = item
		return nil
	}
	return ErrUserNotFound
}

func (r *MemoryRepository) UpsertTOTPCredential(_ context.Context, record TOTPCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.totpByUserID[record.UserID] = record
	return nil
}

func (r *MemoryRepository) FindTOTPCredentialByUserID(_ context.Context, userID uint64) (TOTPCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.totpByUserID[userID]
	if !ok {
		return TOTPCredential{}, ErrUserNotFound
	}
	return record, nil
}

func (r *MemoryRepository) SaveMFAChallenge(_ context.Context, record MFAChallengeRecord) (MFAChallengeRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if record.ID == 0 {
		record.ID = r.nextMFAID
		r.nextMFAID++
	}
	r.mfaByHash[record.TicketHash] = record
	return record, nil
}

func (r *MemoryRepository) FindMFAChallengeByTicketHash(_ context.Context, ticketHash string) (MFAChallengeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.mfaByHash[ticketHash]
	if !ok {
		return MFAChallengeRecord{}, ErrUserNotFound
	}
	return record, nil
}

func (r *MemoryRepository) ConsumeMFAChallenge(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, item := range r.mfaByHash {
		if item.ID != id {
			continue
		}
		now := time.Now()
		item.ConsumedAt = &now
		r.mfaByHash[key] = item
		return nil
	}
	return ErrUserNotFound
}

func (r *MemoryRepository) GetProfileSettings(_ context.Context, userID uint64) (ProfileSettings, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.profiles[userID]
	if !ok {
		return ProfileSettings{}, ErrUserNotFound
	}
	return item, nil
}

func (r *MemoryRepository) UpsertProfileSettings(_ context.Context, item ProfileSettings) (ProfileSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.profiles[item.UserID]; ok {
		item.CreatedAt = existing.CreatedAt
	} else {
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	r.profiles[item.UserID] = item
	return item, nil
}

func (r *PersistentRepository) CreateUser(ctx context.Context, user User) (User, error) {
	return r.userStore.CreateUser(ctx, user)
}

func (r *PersistentRepository) FindUserByLogin(ctx context.Context, login string) (User, error) {
	return r.userStore.FindUserByLogin(ctx, login)
}

func (r *PersistentRepository) FindUserByEmail(ctx context.Context, email string) (User, error) {
	return r.userStore.FindUserByEmail(ctx, email)
}

func (r *PersistentRepository) FindUserByID(ctx context.Context, id uint64) (User, error) {
	return r.userStore.FindUserByID(ctx, id)
}

func (r *PersistentRepository) ListUsers(ctx context.Context) ([]User, error) {
	return r.userStore.ListUsers(ctx)
}

func (r *PersistentRepository) UpdateUserRoles(ctx context.Context, id uint64, roles []string) (User, error) {
	return r.userStore.UpdateUserRoles(ctx, id, roles)
}

func (r *PersistentRepository) UpdateUser(ctx context.Context, user User) (User, error) {
	return r.userStore.UpdateUser(ctx, user)
}

func (r *PersistentRepository) UpdateUserPassword(ctx context.Context, id uint64, passwordHash string) error {
	return r.userStore.UpdateUserPassword(ctx, id, passwordHash)
}

func (r *PersistentRepository) UpdateUserVerification(ctx context.Context, id uint64, emailVerified bool, status string) error {
	return r.userStore.UpdateUserVerification(ctx, id, emailVerified, status)
}

func (r *PersistentRepository) UpdateUserEmail(ctx context.Context, id uint64, email string, emailVerified bool) error {
	return r.userStore.UpdateUserEmail(ctx, id, email, emailVerified)
}

func (r *PersistentRepository) RefreshPendingRegistration(ctx context.Context, id uint64, username string, passwordHash string) (User, error) {
	return r.userStore.RefreshPendingRegistration(ctx, id, username, passwordHash)
}

func (r *PersistentRepository) DeleteUser(ctx context.Context, id uint64) error {
	return r.userStore.DeleteUser(ctx, id)
}

func (r *PersistentRepository) SaveRefreshToken(ctx context.Context, record RefreshTokenRecord) error {
	return r.refreshStore.SaveRefreshToken(ctx, record)
}

func (r *PersistentRepository) FindRefreshToken(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
	return r.refreshStore.FindRefreshToken(ctx, tokenHash)
}

func (r *PersistentRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return r.refreshStore.RevokeRefreshToken(ctx, tokenHash)
}

func (r *PersistentRepository) RevokeUserRefreshTokens(ctx context.Context, userID uint64) error {
	return r.refreshStore.RevokeUserRefreshTokens(ctx, userID)
}

func (r *PersistentRepository) SaveEmailVerification(ctx context.Context, record EmailVerificationRecord) (EmailVerificationRecord, error) {
	return r.verificationStore.SaveEmailVerification(ctx, record)
}

func (r *PersistentRepository) FindEmailVerificationByTicketHash(ctx context.Context, ticketHash string) (EmailVerificationRecord, error) {
	return r.verificationStore.FindEmailVerificationByTicketHash(ctx, ticketHash)
}

func (r *PersistentRepository) ConsumeEmailVerification(ctx context.Context, id uint64) error {
	return r.verificationStore.ConsumeEmailVerification(ctx, id)
}

func (r *PersistentRepository) IncrementEmailVerificationAttempts(ctx context.Context, id uint64) error {
	return r.verificationStore.IncrementEmailVerificationAttempts(ctx, id)
}

func (r *PersistentRepository) UpsertTOTPCredential(ctx context.Context, record TOTPCredential) error {
	return r.totpStore.UpsertTOTPCredential(ctx, record)
}

func (r *PersistentRepository) FindTOTPCredentialByUserID(ctx context.Context, userID uint64) (TOTPCredential, error) {
	return r.totpStore.FindTOTPCredentialByUserID(ctx, userID)
}

func (r *PersistentRepository) SaveMFAChallenge(ctx context.Context, record MFAChallengeRecord) (MFAChallengeRecord, error) {
	return r.mfaStore.SaveMFAChallenge(ctx, record)
}

func (r *PersistentRepository) FindMFAChallengeByTicketHash(ctx context.Context, ticketHash string) (MFAChallengeRecord, error) {
	return r.mfaStore.FindMFAChallengeByTicketHash(ctx, ticketHash)
}

func (r *PersistentRepository) ConsumeMFAChallenge(ctx context.Context, id uint64) error {
	return r.mfaStore.ConsumeMFAChallenge(ctx, id)
}

func (r *PersistentRepository) GetProfileSettings(ctx context.Context, userID uint64) (ProfileSettings, error) {
	return r.profileStore.GetProfileSettings(ctx, userID)
}

func (r *PersistentRepository) UpsertProfileSettings(ctx context.Context, item ProfileSettings) (ProfileSettings, error) {
	return r.profileStore.UpsertProfileSettings(ctx, item)
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
