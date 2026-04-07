package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LegacyMailSyncClient interface {
	ListMailbox(ctx context.Context, mailboxName string) ([]LegacyMessageHeader, error)
	GetMessage(ctx context.Context, mailboxName string, messageID string) (LegacyMessage, error)
}

type LegacyClient struct {
	baseURL string
	http    *http.Client
}

type LegacyMessageHeader struct {
	Mailbox     string    `json:"mailbox"`
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	Subject     string    `json:"subject"`
	Date        time.Time `json:"date"`
	PosixMillis int64     `json:"posix-millis"`
	Size        int64     `json:"size"`
	Seen        bool      `json:"seen"`
}

type LegacyMessage struct {
	Mailbox     string                    `json:"mailbox"`
	ID          string                    `json:"id"`
	From        string                    `json:"from"`
	To          []string                  `json:"to"`
	Subject     string                    `json:"subject"`
	Date        time.Time                 `json:"date"`
	PosixMillis int64                     `json:"posix-millis"`
	Size        int64                     `json:"size"`
	Seen        bool                      `json:"seen"`
	Body        *LegacyMessageBody        `json:"body"`
	Header      map[string][]string       `json:"header"`
	Attachments []LegacyMessageAttachment `json:"attachments"`
}

type LegacyMessageBody struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

type LegacyMessageAttachment struct {
	FileName     string `json:"filename"`
	ContentType  string `json:"content-type"`
	DownloadLink string `json:"download-link"`
	ViewLink     string `json:"view-link"`
	MD5          string `json:"md5"`
}

func NewLegacyMailSyncClient(baseURL string) *LegacyClient {
	return &LegacyClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *LegacyClient) ListMailbox(ctx context.Context, mailboxName string) ([]LegacyMessageHeader, error) {
	endpoint := c.baseURL + "/api/v1/mailbox/" + url.QueryEscape(mailboxName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list mailbox failed with status %d", resp.StatusCode)
	}

	var items []LegacyMessageHeader
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *LegacyClient) GetMessage(ctx context.Context, mailboxName string, messageID string) (LegacyMessage, error) {
	endpoint := c.baseURL + "/api/v1/mailbox/" + url.QueryEscape(mailboxName) + "/" + url.QueryEscape(messageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return LegacyMessage{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return LegacyMessage{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return LegacyMessage{}, fmt.Errorf("get message failed with status %d", resp.StatusCode)
	}

	var item LegacyMessage
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return LegacyMessage{}, err
	}
	return item, nil
}
