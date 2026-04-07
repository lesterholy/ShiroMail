package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"shiro-email/backend/internal/modules/portal"
)

type WebhookRepo interface {
	ListWebhooksByUser(ctx context.Context, userID uint64) ([]portal.Webhook, error)
}

type Dispatcher struct {
	repo   WebhookRepo
	client *http.Client
}

type Payload struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

func NewDispatcher(repo WebhookRepo) *Dispatcher {
	return &Dispatcher{
		repo: repo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, userID uint64, event string, data any) {
	webhooks, err := d.repo.ListWebhooksByUser(ctx, userID)
	if err != nil {
		slog.Error("webhook: failed to list webhooks", "userId", userID, "error", err)
		return
	}

	payload := Payload{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook: failed to marshal payload", "error", err)
		return
	}

	for _, wh := range webhooks {
		if !wh.Enabled || !matchesEvent(wh.Events, event) {
			continue
		}
		go d.deliver(ctx, wh, body)
	}
}

func (d *Dispatcher) deliver(ctx context.Context, wh portal.Webhook, body []byte) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.TargetURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("webhook: failed to create request", "webhookId", wh.ID, "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ShiroEmail-Webhook/1.0")

	if wh.SecretPreview != "" {
		signature := signPayload(body, wh.SecretPreview)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("webhook: delivery failed", "webhookId", wh.ID, "url", wh.TargetURL, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("webhook: delivery returned error", "webhookId", wh.ID, "status", resp.StatusCode)
	} else {
		slog.Debug("webhook: delivered", "webhookId", wh.ID, "status", resp.StatusCode)
	}
}

func signPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func matchesEvent(subscribed []string, event string) bool {
	for _, e := range subscribed {
		if e == "*" || e == event {
			return true
		}
	}
	return false
}
