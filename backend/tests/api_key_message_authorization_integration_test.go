package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"shiro-email/backend/internal/bootstrap"
	"shiro-email/backend/internal/modules/ingest"
	"shiro-email/backend/internal/modules/mailbox"
	"shiro-email/backend/internal/modules/portal"
)

func seedOwnedMailboxWithMessage(t *testing.T, username string, withAttachment bool) (http.Handler, *bootstrap.AppState, string, string, string) {
	t.Helper()

	server, state := newTestServerWithState(t)
	ownerToken := registerAndLogin(t, server, username)

	domainName := username + ".messages.test"
	rr := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"`+domainName+`","status":"active"}`, ownerToken)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected domain create to succeed, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	user, err := state.AuthRepo.FindUserByLogin(context.Background(), username)
	if err != nil {
		t.Fatalf("expected seeded auth user, got %v", err)
	}

	targetMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  domainID,
		Domain:    domainName,
		LocalPart: "inbox",
		Address:   "inbox@" + domainName,
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected mailbox seed success, got %v", err)
	}

	if withAttachment {
		rawMessage := "From: sender@example.com\r\nTo: inbox@" + domainName + "\r\nSubject: API Key Attached\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=abc\r\n\r\n--abc\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nmessage body\r\n--abc\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=\"note.txt\"\r\n\r\nattachment body\r\n--abc--\r\n"
		if _, err := state.DirectIngest.Deliver(context.Background(), ingest.InboundEnvelope{
			MailFrom:   "sender@example.com",
			Recipients: []string{targetMailbox.Address},
		}, strings.NewReader(rawMessage)); err != nil {
			t.Fatalf("expected direct ingest success, got %v", err)
		}
	} else {
		if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), targetMailbox.ID, targetMailbox.LocalPart, ingest.ParsedMessage{
			LegacyMailboxKey: targetMailbox.LocalPart,
			LegacyMessageKey: "message-1",
			FromAddr:         "sender@example.com",
			ToAddr:           targetMailbox.Address,
			Subject:          "API Key Message",
			TextPreview:      "hello from api key test",
			HTMLPreview:      "<p>hello from api key test</p>",
			ReceivedAt:       time.Now(),
		}); err != nil {
			t.Fatalf("expected message seed success, got %v", err)
		}
	}

	items, err := state.MessageRepo.ListByMailboxID(context.Background(), targetMailbox.ID)
	if err != nil {
		t.Fatalf("expected stored messages, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly one stored message, got %d", len(items))
	}

	return server, state, strconv.FormatUint(targetMailbox.ID, 10), strconv.FormatUint(items[0].ID, 10), ownerToken
}

func TestAPIKeyCanListMessagesWithMessagesReadScope(t *testing.T) {
	server, state, mailboxID, _, _ := seedOwnedMailboxWithMessage(t, "api-key-message-reader", false)

	apiKey := createAPIKeyPreview(t, state, "api-key-message-reader", portal.CreateAPIKeyInput{
		Name:   "message-reader",
		Scopes: []string{"messages.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
	})

	rr := performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+mailboxID+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected api key message list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"subject":"API Key Message"`) {
		t.Fatalf("expected seeded message in response: %s", rr.Body.String())
	}
}

func TestAPIKeyCannotDownloadAttachmentWithoutAttachmentScope(t *testing.T) {
	server, state, mailboxID, messageID, _ := seedOwnedMailboxWithMessage(t, "api-key-attachment-reader", true)

	apiKey := createAPIKeyPreview(t, state, "api-key-attachment-reader", portal.CreateAPIKeyInput{
		Name:   "message-reader",
		Scopes: []string{"messages.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mailboxes/"+mailboxID+"/messages/"+messageID+"/attachments/0", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected attachment download without attachment scope to return 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPIKeyMessageAccessHonorsDomainBinding(t *testing.T) {
	server, state := newTestServerWithState(t)
	ownerToken := registerAndLogin(t, server, "api-key-message-bound")
	user, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-message-bound")
	if err != nil {
		t.Fatalf("expected auth user lookup success, got %v", err)
	}

	firstRR := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"bound-msg-one.test","status":"active"}`, ownerToken)
	if firstRR.Code != http.StatusCreated {
		t.Fatalf("expected first domain create success, got %d: %s", firstRR.Code, firstRR.Body.String())
	}
	firstDomainID := parseUint64(t, extractJSONScalarField(firstRR.Body.String(), "id"))

	secondRR := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"bound-msg-two.test","status":"active"}`, ownerToken)
	if secondRR.Code != http.StatusCreated {
		t.Fatalf("expected second domain create success, got %d: %s", secondRR.Code, secondRR.Body.String())
	}
	secondDomainID := parseUint64(t, extractJSONScalarField(secondRR.Body.String(), "id"))

	firstMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  firstDomainID,
		Domain:    "bound-msg-one.test",
		LocalPart: "first",
		Address:   "first@bound-msg-one.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected first mailbox seed success, got %v", err)
	}
	secondMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  secondDomainID,
		Domain:    "bound-msg-two.test",
		LocalPart: "second",
		Address:   "second@bound-msg-two.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected second mailbox seed success, got %v", err)
	}

	for _, item := range []mailbox.Mailbox{firstMailbox, secondMailbox} {
		if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), item.ID, item.LocalPart, ingest.ParsedMessage{
			LegacyMailboxKey: item.LocalPart,
			LegacyMessageKey: "seed-" + item.LocalPart,
			FromAddr:         "sender@example.com",
			ToAddr:           item.Address,
			Subject:          "Bound " + item.LocalPart,
			TextPreview:      "body",
			HTMLPreview:      "<p>body</p>",
			ReceivedAt:       time.Now(),
		}); err != nil {
			t.Fatalf("expected message seed success for %s, got %v", item.Address, err)
		}
	}

	boundNodeID := firstDomainID
	apiKey := createAPIKeyPreview(t, state, "api-key-message-bound", portal.CreateAPIKeyInput{
		Name:   "bound-message-reader",
		Scopes: []string{"messages.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
		DomainBindings: []portal.APIKeyDomainBinding{
			{
				NodeID:      &boundNodeID,
				AccessLevel: "read",
			},
		},
	})

	rr := performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(firstMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected bound mailbox messages list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(secondMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected unbound mailbox messages list to return 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAPIKeyWithoutBindingsCanReadMessagesAcrossOwnedMailboxes(t *testing.T) {
	server, state := newTestServerWithState(t)
	ownerToken := registerAndLogin(t, server, "api-key-message-unbound")
	user, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-message-unbound")
	if err != nil {
		t.Fatalf("expected auth user lookup success, got %v", err)
	}

	firstRR := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"unbound-msg-one.test","status":"active"}`, ownerToken)
	if firstRR.Code != http.StatusCreated {
		t.Fatalf("expected first domain create success, got %d: %s", firstRR.Code, firstRR.Body.String())
	}
	firstDomainID := parseUint64(t, extractJSONScalarField(firstRR.Body.String(), "id"))

	secondRR := performJSON(server, http.MethodPost, "/api/v1/domains", `{"domain":"unbound-msg-two.test","status":"active"}`, ownerToken)
	if secondRR.Code != http.StatusCreated {
		t.Fatalf("expected second domain create success, got %d: %s", secondRR.Code, secondRR.Body.String())
	}
	secondDomainID := parseUint64(t, extractJSONScalarField(secondRR.Body.String(), "id"))

	firstMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  firstDomainID,
		Domain:    "unbound-msg-one.test",
		LocalPart: "first",
		Address:   "first@unbound-msg-one.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected first mailbox seed success, got %v", err)
	}
	secondMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  secondDomainID,
		Domain:    "unbound-msg-two.test",
		LocalPart: "second",
		Address:   "second@unbound-msg-two.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected second mailbox seed success, got %v", err)
	}

	for _, item := range []mailbox.Mailbox{firstMailbox, secondMailbox} {
		if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), item.ID, item.LocalPart, ingest.ParsedMessage{
			LegacyMailboxKey: item.LocalPart,
			LegacyMessageKey: "seed-" + item.LocalPart,
			FromAddr:         "sender@example.com",
			ToAddr:           item.Address,
			Subject:          "Unbound " + item.LocalPart,
			TextPreview:      "body",
			HTMLPreview:      "<p>body</p>",
			ReceivedAt:       time.Now(),
		}); err != nil {
			t.Fatalf("expected message seed success for %s, got %v", item.Address, err)
		}
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-message-unbound", portal.CreateAPIKeyInput{
		Name:   "unbound-message-reader",
		Scopes: []string{"messages.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode: "public_only",
		},
	})

	rr := performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(firstMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected first owned mailbox messages list to return 200 without bindings, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"subject":"Unbound first"`) {
		t.Fatalf("expected first seeded message in response: %s", rr.Body.String())
	}

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(secondMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected second owned mailbox messages list to return 200 without bindings, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"subject":"Unbound second"`) {
		t.Fatalf("expected second seeded message in response: %s", rr.Body.String())
	}
}

func TestAPIKeyWithoutBindingsCannotReadMessagesFromInaccessiblePrivateMailbox(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)
	registerAndLogin(t, server, "api-key-message-private-owner")
	registerAndLogin(t, server, "api-key-message-inaccessible")

	owner, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-message-private-owner")
	if err != nil {
		t.Fatalf("expected private owner lookup success, got %v", err)
	}

	if _, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-message-inaccessible"); err != nil {
		t.Fatalf("expected auth user lookup success, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"inaccessible-message-domain.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create success, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	targetMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    owner.ID,
		DomainID:  domainID,
		Domain:    "inaccessible-message-domain.test",
		LocalPart: "hidden",
		Address:   "hidden@inaccessible-message-domain.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected mailbox seed success, got %v", err)
	}

	if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), targetMailbox.ID, targetMailbox.LocalPart, ingest.ParsedMessage{
		LegacyMailboxKey: targetMailbox.LocalPart,
		LegacyMessageKey: "hidden-message",
		FromAddr:         "sender@example.com",
		ToAddr:           targetMailbox.Address,
		Subject:          "Hidden message",
		TextPreview:      "body",
		HTMLPreview:      "<p>body</p>",
		ReceivedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("expected message seed success, got %v", err)
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-message-inaccessible", portal.CreateAPIKeyInput{
		Name:   "unbound-hidden-message-reader",
		Scopes: []string{"messages.read", "mailboxes.read"},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(targetMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected inaccessible private mailbox messages list to return 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminAPIKeyCanListMessagesOnBoundPlatformPrivateDomain(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)

	adminUser, err := state.AuthRepo.FindUserByLogin(context.Background(), "admin")
	if err != nil {
		t.Fatalf("expected seeded admin user lookup success, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"admin-private-messages.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create success, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	targetMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    adminUser.ID,
		DomainID:  domainID,
		Domain:    "admin-private-messages.test",
		LocalPart: "ops",
		Address:   "ops@admin-private-messages.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected admin mailbox seed success, got %v", err)
	}

	if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), targetMailbox.ID, targetMailbox.LocalPart, ingest.ParsedMessage{
		LegacyMailboxKey: targetMailbox.LocalPart,
		LegacyMessageKey: "admin-platform-bound-message",
		FromAddr:         "sender@example.com",
		ToAddr:           targetMailbox.Address,
		Subject:          "Admin Bound Message",
		TextPreview:      "admin bound body",
		HTMLPreview:      "<p>admin bound body</p>",
		ReceivedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("expected message seed success, got %v", err)
	}

	boundNodeID := domainID
	apiKey := createAPIKeyPreview(t, state, "admin", portal.CreateAPIKeyInput{
		Name:   "admin-bound-message-reader",
		Scopes: []string{"messages.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode:         "private_only",
			AllowOwnedPrivateDomains: true,
		},
		DomainBindings: []portal.APIKeyDomainBinding{
			{
				NodeID:      &boundNodeID,
				AccessLevel: "read",
			},
		},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(targetMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected bound admin mailbox messages list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"subject":"Admin Bound Message"`) {
		t.Fatalf("expected seeded admin message in response: %s", rr.Body.String())
	}
}

func TestAdminAPIKeyWithoutBindingsCanReadPlatformPrivateMessages(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)

	adminUser, err := state.AuthRepo.FindUserByLogin(context.Background(), "admin")
	if err != nil {
		t.Fatalf("expected seeded admin user lookup success, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"admin-unbound-private-messages.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create success, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	targetMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    adminUser.ID,
		DomainID:  domainID,
		Domain:    "admin-unbound-private-messages.test",
		LocalPart: "ops",
		Address:   "ops@admin-unbound-private-messages.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected admin mailbox seed success, got %v", err)
	}

	if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), targetMailbox.ID, targetMailbox.LocalPart, ingest.ParsedMessage{
		LegacyMailboxKey: targetMailbox.LocalPart,
		LegacyMessageKey: "admin-platform-unbound-message",
		FromAddr:         "sender@example.com",
		ToAddr:           targetMailbox.Address,
		Subject:          "Admin Unbound Message",
		TextPreview:      "admin unbound body",
		HTMLPreview:      "<p>admin unbound body</p>",
		ReceivedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("expected message seed success, got %v", err)
	}

	apiKey := createAPIKeyPreview(t, state, "admin", portal.CreateAPIKeyInput{
		Name:   "admin-unbound-message-reader",
		Scopes: []string{"messages.read"},
		ResourcePolicy: portal.APIKeyResourcePolicy{
			DomainAccessMode: "private_only",
		},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(targetMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected unbound admin mailbox messages list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"subject":"Admin Unbound Message"`) {
		t.Fatalf("expected seeded admin message in response, got %s", rr.Body.String())
	}
}

func TestUnboundAPIKeyCanReadOwnMessagesOnPlatformPrivateDomain(t *testing.T) {
	server, state := newTestServerWithState(t)
	adminToken := adminAccessToken(t)

	registerAndLogin(t, server, "api-key-own-platform-message")
	user, err := state.AuthRepo.FindUserByLogin(context.Background(), "api-key-own-platform-message")
	if err != nil {
		t.Fatalf("expected seeded user lookup success, got %v", err)
	}

	rr := performJSON(server, http.MethodPost, "/api/v1/admin/domains", `{"domain":"own-platform-message.test","status":"active","visibility":"private"}`, adminToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected admin private domain create success, got %d: %s", rr.Code, rr.Body.String())
	}
	domainID := parseUint64(t, extractJSONScalarField(rr.Body.String(), "id"))

	targetMailbox, err := state.MailboxRepo.Create(context.Background(), mailbox.Mailbox{
		UserID:    user.ID,
		DomainID:  domainID,
		Domain:    "own-platform-message.test",
		LocalPart: "owned",
		Address:   "owned@own-platform-message.test",
		Status:    "active",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected mailbox seed success, got %v", err)
	}

	if err := state.MessageRepo.UpsertFromLegacySync(context.Background(), targetMailbox.ID, targetMailbox.LocalPart, ingest.ParsedMessage{
		LegacyMailboxKey: targetMailbox.LocalPart,
		LegacyMessageKey: "own-platform-message",
		FromAddr:         "sender@example.com",
		ToAddr:           targetMailbox.Address,
		Subject:          "Own Platform Message",
		TextPreview:      "body",
		HTMLPreview:      "<p>body</p>",
		ReceivedAt:       time.Now(),
	}); err != nil {
		t.Fatalf("expected message seed success, got %v", err)
	}

	apiKey := createAPIKeyPreview(t, state, "api-key-own-platform-message", portal.CreateAPIKeyInput{
		Name:   "own-platform-message-reader",
		Scopes: []string{"mailboxes.read", "messages.read"},
	})

	rr = performJSON(server, http.MethodGet, "/api/v1/mailboxes/"+strconv.FormatUint(targetMailbox.ID, 10)+"/messages", "", apiKey)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected own platform private mailbox messages list to return 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"subject":"Own Platform Message"`) {
		t.Fatalf("expected own platform private message in response, got %s", rr.Body.String())
	}
}
