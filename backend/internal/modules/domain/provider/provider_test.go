package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResolveSecretRefReadsEnvVariable(t *testing.T) {
	t.Setenv("SHIRO_PROVIDER_SECRET", `{"apiKey":"key-123","apiSecret":"secret-456"}`)

	resolved, err := ResolveSecretRef("env:SHIRO_PROVIDER_SECRET")
	if err != nil {
		t.Fatalf("expected env secret ref to resolve, got %v", err)
	}
	if resolved != `{"apiKey":"key-123","apiSecret":"secret-456"}` {
		t.Fatalf("expected env secret payload, got %q", resolved)
	}
}

func TestResolveSecretRefReturnsInlineSecret(t *testing.T) {
	resolved, err := ResolveSecretRef(`{"apiToken":"cf-token"}`)
	if err != nil {
		t.Fatalf("expected inline secret ref to resolve, got %v", err)
	}
	if resolved != `{"apiToken":"cf-token"}` {
		t.Fatalf("expected inline secret payload, got %q", resolved)
	}
}

func TestResolveCloudflareCredentialsSupportsGlobalAPIKey(t *testing.T) {
	credentials, err := ResolveCloudflareCredentials(`{"email":"ops@example.com","apiKey":"global-key"}`)
	if err != nil {
		t.Fatalf("expected cloudflare api key credentials to resolve, got %v", err)
	}
	if credentials.Email != "ops@example.com" || credentials.APIKey != "global-key" {
		t.Fatalf("expected parsed cloudflare api key credentials, got %#v", credentials)
	}
}

func TestCloudflareValidateUsesVerifyEndpointWithBearerToken(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"success":true,"result":{"status":"active"}}`),
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	result, err := adapter.Validate(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_token",
		SecretRef: `{"apiToken":"cf-token"}`,
	})
	if err != nil {
		t.Fatalf("expected validate request to succeed, got %v", err)
	}
	if result.Status != "healthy" {
		t.Fatalf("expected healthy provider validation result, got %q", result.Status)
	}
	if transport.request == nil {
		t.Fatal("expected validate request to be captured")
	}
	if transport.request.Method != http.MethodGet {
		t.Fatalf("expected GET request, got %s", transport.request.Method)
	}
	if transport.request.URL.Path != "/client/v4/user/tokens/verify" {
		t.Fatalf("expected verify endpoint, got %s", transport.request.URL.Path)
	}
	if got := transport.request.Header.Get("Authorization"); got != "Bearer cf-token" {
		t.Fatalf("expected bearer token header, got %q", got)
	}
}

func TestCloudflareValidateUsesUserEndpointWithGlobalAPIKey(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"success":true,"result":{"id":"user-1","email":"ops@example.com"}}`),
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	result, err := adapter.Validate(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_key",
		SecretRef: `{"email":"ops@example.com","apiKey":"global-key"}`,
	})
	if err != nil {
		t.Fatalf("expected validate request with global api key to succeed, got %v", err)
	}
	if result.Status != "healthy" {
		t.Fatalf("expected healthy provider validation result, got %q", result.Status)
	}
	if transport.request == nil {
		t.Fatal("expected validate request to be captured")
	}
	if transport.request.URL.Path != "/client/v4/user" {
		t.Fatalf("expected user endpoint for global api key validation, got %s", transport.request.URL.Path)
	}
	if got := transport.request.Header.Get("X-Auth-Email"); got != "ops@example.com" {
		t.Fatalf("expected X-Auth-Email header, got %q", got)
	}
	if got := transport.request.Header.Get("X-Auth-Key"); got != "global-key" {
		t.Fatalf("expected X-Auth-Key header, got %q", got)
	}
	if got := transport.request.Header.Get("Authorization"); got != "" {
		t.Fatalf("expected no Authorization header for global api key validation, got %q", got)
	}
}

func TestCloudflareListZonesUsesZonesEndpointWithBearerToken(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"success":true,"result":[{"id":"zone-1","name":"example.com","status":"active"}]}`),
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	zones, err := adapter.ListZones(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_token",
		SecretRef: "cf-token",
	})
	if err != nil {
		t.Fatalf("expected list zones request to succeed, got %v", err)
	}
	if len(zones) != 1 || zones[0].Name != "example.com" {
		t.Fatalf("expected parsed cloudflare zone, got %#v", zones)
	}
	if transport.request == nil {
		t.Fatal("expected zones request to be captured")
	}
	if transport.request.Method != http.MethodGet {
		t.Fatalf("expected GET request, got %s", transport.request.Method)
	}
	if transport.request.URL.Path != "/client/v4/zones" {
		t.Fatalf("expected zones endpoint, got %s", transport.request.URL.Path)
	}
	if got := transport.request.URL.Query().Get("page"); got != "1" {
		t.Fatalf("expected page query param 1, got %q", got)
	}
	if got := transport.request.URL.Query().Get("per_page"); got != "100" {
		t.Fatalf("expected per_page query param 100, got %q", got)
	}
	if got := transport.request.Header.Get("Authorization"); got != "Bearer cf-token" {
		t.Fatalf("expected bearer token header, got %q", got)
	}
}

func TestCloudflareListZonesUsesGlobalAPIKeyHeaders(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"success":true,"result":[{"id":"zone-1","name":"example.com","status":"active"}]}`),
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	zones, err := adapter.ListZones(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_key",
		SecretRef: `{"email":"ops@example.com","apiKey":"global-key"}`,
	})
	if err != nil {
		t.Fatalf("expected list zones with global api key to succeed, got %v", err)
	}
	if len(zones) != 1 || zones[0].Name != "example.com" {
		t.Fatalf("expected parsed cloudflare zone, got %#v", zones)
	}
	if transport.request == nil {
		t.Fatal("expected zones request to be captured")
	}
	if got := transport.request.Header.Get("X-Auth-Email"); got != "ops@example.com" {
		t.Fatalf("expected X-Auth-Email header, got %q", got)
	}
	if got := transport.request.Header.Get("X-Auth-Key"); got != "global-key" {
		t.Fatalf("expected X-Auth-Key header, got %q", got)
	}
}

func TestCloudflareListZonesReturnsProviderErrors(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"success":false,"errors":[{"code":6003,"message":"Invalid request headers"}],"result":null}`),
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	_, err := adapter.ListZones(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_token",
		SecretRef: "cf-token",
	})
	if err == nil || !strings.Contains(err.Error(), "6003") {
		t.Fatalf("expected surfaced cloudflare provider error, got %v", err)
	}
}

func TestDecodeJSONResponseReturnsReadableProviderErrorBody(t *testing.T) {
	response := jsonResponse(http.StatusBadRequest, `{"errors":[{"code":6003,"message":"Invalid request headers"}]}`)

	var payload map[string]any
	err := decodeJSONResponse(response, &payload)
	if err == nil {
		t.Fatal("expected decodeJSONResponse to return provider error")
	}
	if !strings.Contains(err.Error(), "Invalid request headers") {
		t.Fatalf("expected readable provider error, got %v", err)
	}
}

func TestCloudflareListRecordsUsesDNSRecordsEndpointWithBearerToken(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"success":true,"result":[{"type":"MX","name":"example.com","content":"mx1.example.com","ttl":120,"priority":10,"proxied":false}]}`),
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	records, err := adapter.ListRecords(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_token",
		SecretRef: "cf-token",
	}, "zone-1")
	if err != nil {
		t.Fatalf("expected list records request to succeed, got %v", err)
	}
	if len(records) != 1 || records[0].Value != "mx1.example.com" {
		t.Fatalf("expected parsed cloudflare records, got %#v", records)
	}
	if transport.request == nil {
		t.Fatal("expected records request to be captured")
	}
	if transport.request.Method != http.MethodGet {
		t.Fatalf("expected GET request, got %s", transport.request.Method)
	}
	if transport.request.URL.Path != "/client/v4/zones/zone-1/dns_records" {
		t.Fatalf("expected dns records endpoint, got %s", transport.request.URL.Path)
	}
	if got := transport.request.URL.Query().Get("page"); got != "1" {
		t.Fatalf("expected page query param 1, got %q", got)
	}
	if got := transport.request.URL.Query().Get("per_page"); got != "100" {
		t.Fatalf("expected per_page query param 100, got %q", got)
	}
	if got := transport.request.Header.Get("Authorization"); got != "Bearer cf-token" {
		t.Fatalf("expected bearer token header, got %q", got)
	}
}

func TestSpaceshipValidateUsesDomainListEndpointWithAPIHeaders(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"items":[{"name":"example.com","status":"active"}]}`),
	}
	adapter := NewSpaceshipAdapter("https://spaceship.test/api", &http.Client{Transport: transport})

	result, err := adapter.Validate(context.Background(), Account{
		Provider:  "spaceship",
		AuthType:  "api_key",
		SecretRef: `{"apiKey":"ship-key","apiSecret":"ship-secret"}`,
	})
	if err != nil {
		t.Fatalf("expected validate request to succeed, got %v", err)
	}
	if result.Status != "healthy" {
		t.Fatalf("expected healthy provider validation result, got %q", result.Status)
	}
	if transport.request == nil {
		t.Fatal("expected validate request to be captured")
	}
	if transport.request.Method != http.MethodGet {
		t.Fatalf("expected GET request, got %s", transport.request.Method)
	}
	if transport.request.URL.Path != "/api/v1/domains" {
		t.Fatalf("expected domains endpoint, got %s", transport.request.URL.Path)
	}
	if got := transport.request.URL.Query().Get("take"); got != "1" {
		t.Fatalf("expected take query param 1, got %q", got)
	}
	if got := transport.request.URL.Query().Get("skip"); got != "0" {
		t.Fatalf("expected skip query param 0, got %q", got)
	}
	if got := transport.request.Header.Get("X-API-Key"); got != "ship-key" {
		t.Fatalf("expected X-API-Key header, got %q", got)
	}
	if got := transport.request.Header.Get("X-API-Secret"); got != "ship-secret" {
		t.Fatalf("expected X-API-Secret header, got %q", got)
	}
}

func TestSpaceshipListZonesUsesDomainListEndpointWithPagination(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"items":[{"name":"example.com","status":"active"},{"name":"relay.test","status":"active"}]}`),
	}
	adapter := NewSpaceshipAdapter("https://spaceship.test/api", &http.Client{Transport: transport})

	zones, err := adapter.ListZones(context.Background(), Account{
		Provider:  "spaceship",
		AuthType:  "api_key",
		SecretRef: `{"apiKey":"ship-key","apiSecret":"ship-secret"}`,
	})
	if err != nil {
		t.Fatalf("expected list zones request to succeed, got %v", err)
	}
	if len(zones) != 2 || zones[1].Name != "relay.test" {
		t.Fatalf("expected parsed spaceship zones, got %#v", zones)
	}
	if transport.request == nil {
		t.Fatal("expected zones request to be captured")
	}
	if transport.request.URL.Path != "/api/v1/domains" {
		t.Fatalf("expected domains endpoint, got %s", transport.request.URL.Path)
	}
	if got := transport.request.URL.Query().Get("take"); got != "100" {
		t.Fatalf("expected take query param 100, got %q", got)
	}
	if got := transport.request.URL.Query().Get("skip"); got != "0" {
		t.Fatalf("expected skip query param 0, got %q", got)
	}
}

func TestSpaceshipListRecordsUsesDomainRecordsEndpoint(t *testing.T) {
	transport := &captureRoundTripper{
		response: jsonResponse(http.StatusOK, `{"items":[{"type":"TXT","name":"_dmarc","value":"v=DMARC1; p=none"},{"type":"MX","name":"@","exchange":"mx1.example.com","preference":10},{"type":"CNAME","name":"mail","cname":"smtp.example.com"}]}`),
	}
	adapter := NewSpaceshipAdapter("https://spaceship.test/api", &http.Client{Transport: transport})

	records, err := adapter.ListRecords(context.Background(), Account{
		Provider:  "spaceship",
		AuthType:  "api_key",
		SecretRef: `{"apiKey":"ship-key","apiSecret":"ship-secret"}`,
	}, "example.com")
	if err != nil {
		t.Fatalf("expected list records request to succeed, got %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected parsed spaceship records, got %#v", records)
	}
	if records[0].Name != "_dmarc.example.com" {
		t.Fatalf("expected spaceship record name to be normalized to fqdn, got %#v", records[0])
	}
	if records[1].Name != "example.com" || records[1].Value != "mx1.example.com" || records[1].Priority != 10 {
		t.Fatalf("expected spaceship MX record aliases to be normalized, got %#v", records[1])
	}
	if records[2].Name != "mail.example.com" || records[2].Value != "smtp.example.com" {
		t.Fatalf("expected spaceship CNAME alias to be normalized, got %#v", records[2])
	}
	if transport.request == nil {
		t.Fatal("expected records request to be captured")
	}
	if transport.request.URL.Path != "/api/v1/dns/records/example.com" {
		t.Fatalf("expected dns records endpoint, got %s", transport.request.URL.Path)
	}
	if got := transport.request.URL.Query().Get("take"); got != "100" {
		t.Fatalf("expected take query param 100, got %q", got)
	}
	if got := transport.request.URL.Query().Get("skip"); got != "0" {
		t.Fatalf("expected skip query param 0, got %q", got)
	}
}

func TestCloudflareApplyChangesUsesCreateUpdateDeleteEndpoints(t *testing.T) {
	transport := &recordingRoundTripper{
		responder: func(request *http.Request, _ string) *http.Response {
			switch {
			case request.Method == http.MethodPost && request.URL.Path == "/client/v4/zones/zone-1/dns_records":
				return jsonResponse(http.StatusOK, `{"success":true,"result":{"id":"created-record"}}`)
			case request.Method == http.MethodPatch && request.URL.Path == "/client/v4/zones/zone-1/dns_records/record-update":
				return jsonResponse(http.StatusOK, `{"success":true,"result":{"id":"record-update"}}`)
			case request.Method == http.MethodDelete && request.URL.Path == "/client/v4/zones/zone-1/dns_records/record-delete":
				return jsonResponse(http.StatusOK, `{"success":true,"result":{"id":"record-delete"}}`)
			default:
				return jsonResponse(http.StatusNotFound, `{"success":false}`)
			}
		},
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	err := adapter.ApplyChanges(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_token",
		SecretRef: `{"apiToken":"cf-token"}`,
	}, "zone-1", []Change{
		{
			Operation: "create",
			After: &Record{
				Type:    "A",
				Name:    "mail.example.com",
				Value:   "1.2.3.4",
				TTL:     120,
				Proxied: true,
			},
		},
		{
			Operation: "update",
			Before: &Record{
				ID:    "record-update",
				Type:  "TXT",
				Name:  "_dmarc.example.com",
				Value: "v=DMARC1; p=none",
				TTL:   120,
			},
			After: &Record{
				Type:  "TXT",
				Name:  "_dmarc.example.com",
				Value: "v=DMARC1; p=quarantine",
				TTL:   300,
			},
		},
		{
			Operation: "delete",
			Before: &Record{
				ID:       "record-delete",
				Type:     "MX",
				Name:     "example.com",
				Value:    "mx1.example.com",
				TTL:      120,
				Priority: 10,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected cloudflare apply changes to succeed, got %v", err)
	}
	if len(transport.requests) != 3 {
		t.Fatalf("expected 3 provider write requests, got %#v", transport.requests)
	}

	createRequest := transport.requests[0]
	if createRequest.Method != http.MethodPost || createRequest.Path != "/client/v4/zones/zone-1/dns_records" {
		t.Fatalf("expected create request to hit dns_records endpoint, got %#v", createRequest)
	}
	if got := createRequest.Headers.Get("Authorization"); got != "Bearer cf-token" {
		t.Fatalf("expected bearer token on create request, got %q", got)
	}
	if !strings.Contains(createRequest.Body, `"type":"A"`) ||
		!strings.Contains(createRequest.Body, `"name":"mail.example.com"`) ||
		!strings.Contains(createRequest.Body, `"content":"1.2.3.4"`) ||
		!strings.Contains(createRequest.Body, `"proxied":true`) {
		t.Fatalf("expected create request body to contain Cloudflare DNS payload, got %s", createRequest.Body)
	}

	updateRequest := transport.requests[1]
	if updateRequest.Method != http.MethodPatch || updateRequest.Path != "/client/v4/zones/zone-1/dns_records/record-update" {
		t.Fatalf("expected update request to hit patch endpoint, got %#v", updateRequest)
	}
	if !strings.Contains(updateRequest.Body, `"content":"v=DMARC1; p=quarantine"`) || !strings.Contains(updateRequest.Body, `"ttl":300`) {
		t.Fatalf("expected update request body to contain patched payload, got %s", updateRequest.Body)
	}

	deleteRequest := transport.requests[2]
	if deleteRequest.Method != http.MethodDelete || deleteRequest.Path != "/client/v4/zones/zone-1/dns_records/record-delete" {
		t.Fatalf("expected delete request to hit delete endpoint, got %#v", deleteRequest)
	}
	if deleteRequest.Body != "" {
		t.Fatalf("expected delete request body to be empty, got %q", deleteRequest.Body)
	}
}

func TestCloudflareApplyChangesReturnsProviderErrorsOnSuccessfulHTTPStatus(t *testing.T) {
	transport := &recordingRoundTripper{
		responder: func(request *http.Request, _ string) *http.Response {
			return jsonResponse(http.StatusOK, `{"success":false,"errors":[{"code":81057,"message":"record already exists"}]}`)
		},
	}
	adapter := NewCloudflareAdapter("https://cf.test/client/v4", &http.Client{Transport: transport})

	err := adapter.ApplyChanges(context.Background(), Account{
		Provider:  "cloudflare",
		AuthType:  "api_token",
		SecretRef: `{"apiToken":"cf-token"}`,
	}, "zone-1", []Change{
		{
			Operation: "create",
			After: &Record{
				Type:  "TXT",
				Name:  "_acme-challenge.example.com",
				Value: "token-value",
				TTL:   120,
			},
		},
	})
	if err == nil {
		t.Fatal("expected cloudflare apply changes to surface provider error")
	}
	if !strings.Contains(err.Error(), "81057") || !strings.Contains(err.Error(), "record already exists") {
		t.Fatalf("expected surfaced cloudflare provider error, got %v", err)
	}
}

func TestSpaceshipApplyChangesUsesSaveAndDeleteRecordEndpoints(t *testing.T) {
	transport := &recordingRoundTripper{
		responder: func(request *http.Request, _ string) *http.Response {
			switch {
			case request.Method == http.MethodPut && request.URL.Path == "/api/v1/dns/records/example.com":
				return jsonResponse(http.StatusNoContent, "")
			case request.Method == http.MethodDelete && request.URL.Path == "/api/v1/dns/records/example.com":
				return jsonResponse(http.StatusNoContent, "")
			default:
				return jsonResponse(http.StatusNotFound, `{"detail":"not found"}`)
			}
		},
	}
	adapter := NewSpaceshipAdapter("https://spaceship.test/api", &http.Client{Transport: transport})

	err := adapter.ApplyChanges(context.Background(), Account{
		Provider:  "spaceship",
		AuthType:  "api_key",
		SecretRef: `{"apiKey":"ship-key","apiSecret":"ship-secret"}`,
	}, "example.com", []Change{
		{
			Operation: "create",
			After: &Record{
				Type:  "A",
				Name:  "example.com",
				Value: "1.2.3.4",
				TTL:   600,
			},
		},
		{
			Operation: "update",
			Before: &Record{
				Type:  "TXT",
				Name:  "_dmarc.example.com",
				Value: "v=DMARC1; p=none",
				TTL:   120,
			},
			After: &Record{
				Type:  "TXT",
				Name:  "_dmarc.example.com",
				Value: "v=DMARC1; p=quarantine",
				TTL:   300,
			},
		},
		{
			Operation: "create",
			After: &Record{
				Type:  "CNAME",
				Name:  "mail.example.com",
				Value: "smtp.example.com",
				TTL:   300,
			},
		},
		{
			Operation: "delete",
			Before: &Record{
				Type:     "MX",
				Name:     "example.com",
				Value:    "mx1.example.com",
				Priority: 10,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected spaceship apply changes to succeed, got %v", err)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("expected 2 spaceship provider write requests, got %#v", transport.requests)
	}

	saveRequest := transport.requests[0]
	if saveRequest.Method != http.MethodPut || saveRequest.Path != "/api/v1/dns/records/example.com" {
		t.Fatalf("expected save request to hit PUT dns endpoint, got %#v", saveRequest)
	}
	if got := saveRequest.Headers.Get("X-API-Key"); got != "ship-key" {
		t.Fatalf("expected X-API-Key header, got %q", got)
	}
	if got := saveRequest.Headers.Get("X-API-Secret"); got != "ship-secret" {
		t.Fatalf("expected X-API-Secret header, got %q", got)
	}
	if !strings.Contains(saveRequest.Body, `"force":false`) ||
		!strings.Contains(saveRequest.Body, `"items":[`) ||
		!strings.Contains(saveRequest.Body, `"type":"A"`) ||
		!strings.Contains(saveRequest.Body, `"address":"1.2.3.4"`) ||
		!strings.Contains(saveRequest.Body, `"name":"@"`) ||
		!strings.Contains(saveRequest.Body, `"type":"TXT"`) ||
		!strings.Contains(saveRequest.Body, `"name":"_dmarc"`) ||
		!strings.Contains(saveRequest.Body, `"value":"v=DMARC1; p=quarantine"`) {
		t.Fatalf("expected save request to contain create/update payloads, got %s", saveRequest.Body)
	}
	if strings.Contains(saveRequest.Body, `"target":"`) {
		t.Fatalf("expected spaceship CNAME payload to avoid target alias, got %s", saveRequest.Body)
	}
	if !strings.Contains(saveRequest.Body, `"cname":"`) {
		t.Fatalf("expected spaceship CNAME payload to use cname field, got %s", saveRequest.Body)
	}

	deleteRequest := transport.requests[1]
	if deleteRequest.Method != http.MethodDelete || deleteRequest.Path != "/api/v1/dns/records/example.com" {
		t.Fatalf("expected delete request to hit DELETE dns endpoint, got %#v", deleteRequest)
	}
	if strings.Contains(deleteRequest.Body, `"items"`) ||
		!strings.Contains(deleteRequest.Body, `"type":"MX"`) ||
		!strings.Contains(deleteRequest.Body, `"name":"@"`) ||
		!strings.Contains(deleteRequest.Body, `"exchange":"mx1.example.com"`) ||
		!strings.Contains(deleteRequest.Body, `"preference":10`) {
		t.Fatalf("expected delete request to contain MX delete payload, got %s", deleteRequest.Body)
	}
}

type captureRoundTripper struct {
	request  *http.Request
	body     string
	response *http.Response
	err      error
}

func (c *captureRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	c.request = request.Clone(request.Context())
	if request.Body != nil {
		payload, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			return nil, readErr
		}
		c.body = string(payload)
	}
	return c.response, c.err
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

type recordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    string
}

type recordingRoundTripper struct {
	requests  []recordedRequest
	responder func(request *http.Request, body string) *http.Response
}

func (r *recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	body := ""
	if request.Body != nil {
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		body = string(payload)
	}
	r.requests = append(r.requests, recordedRequest{
		Method:  request.Method,
		Path:    request.URL.Path,
		Headers: request.Header.Clone(),
		Body:    body,
	})
	if r.responder == nil {
		return jsonResponse(http.StatusOK, `{"success":true}`), nil
	}
	return r.responder(request, body), nil
}
