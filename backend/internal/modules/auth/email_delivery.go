package auth

import (
	"context"

	"shiro-email/backend/internal/modules/system"
)

type EmailSender interface {
	SendVerificationCode(ctx context.Context, to string, code string, actionURL string) error
	SendPasswordResetCode(ctx context.Context, to string, code string, actionURL string) error
}

type ConfigSMTPEmailSender struct {
	configRepo system.ConfigRepository
}

type NoopEmailSender struct{}

func NewConfigSMTPEmailSender(configRepo system.ConfigRepository) *ConfigSMTPEmailSender {
	return &ConfigSMTPEmailSender{configRepo: configRepo}
}

func NewNoopEmailSender() *NoopEmailSender {
	return &NoopEmailSender{}
}

func (s *NoopEmailSender) SendVerificationCode(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (s *NoopEmailSender) SendPasswordResetCode(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (s *ConfigSMTPEmailSender) SendVerificationCode(ctx context.Context, to string, code string, actionURL string) error {
	return s.sendCodeEmail(ctx, to, code, "Shiro Email verification code", "Your verification code is:", actionURL, "Verify email")
}

func (s *ConfigSMTPEmailSender) SendPasswordResetCode(ctx context.Context, to string, code string, actionURL string) error {
	return s.sendCodeEmail(ctx, to, code, "Shiro Email password reset code", "Your password reset code is:", actionURL, "Reset password")
}

func (s *ConfigSMTPEmailSender) sendCodeEmail(ctx context.Context, to string, code string, subject string, intro string, actionURL string, actionLabel string) error {
	settings, err := system.LoadMailDeliverySettings(ctx, s.configRepo)
	if err != nil {
		return err
	}
	return system.SendMailDeliveryCode(settings, to, subject, intro, code, actionURL, actionLabel)
}
