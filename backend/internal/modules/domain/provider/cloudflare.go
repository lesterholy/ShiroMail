package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultCloudflareBaseURL = "https://api.cloudflare.com/client/v4"

type CloudflareAdapter struct {
	baseURL string
	client  *http.Client
}

type cloudflareAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewCloudflareAdapter(baseURL string, client *http.Client) *CloudflareAdapter {
	if baseURL == "" {
		baseURL = defaultCloudflareBaseURL
	}
	return &CloudflareAdapter{
		baseURL: baseURL,
		client:  defaultHTTPClient(client),
	}
}

func (a *CloudflareAdapter) Validate(ctx context.Context, account Account) (ValidationResult, error) {
	credentials, err := ResolveCloudflareCredentials(account.SecretRef)
	if err != nil {
		return ValidationResult{}, err
	}

	validationPath := "/user/tokens/verify"
	switch strings.TrimSpace(strings.ToLower(account.AuthType)) {
	case "api_key", "global_api_key":
		validationPath = "/user"
	}

	endpoint, err := joinURL(a.baseURL, validationPath)
	if err != nil {
		return ValidationResult{}, err
	}

	req, err := newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ValidationResult{}, err
	}
	if err := applyCloudflareAuthHeaders(req, account.AuthType, credentials); err != nil {
		return ValidationResult{}, err
	}

	response, err := a.client.Do(req)
	if err != nil {
		return ValidationResult{}, err
	}

	var payload struct {
		Success bool                 `json:"success"`
		Errors  []cloudflareAPIError `json:"errors"`
		Result  json.RawMessage      `json:"result"`
	}
	if err := decodeJSONResponse(response, &payload); err != nil {
		return ValidationResult{}, err
	}
	if err := ensureCloudflareSuccess(payload.Success, payload.Errors); err != nil {
		return ValidationResult{}, err
	}

	status := "healthy"
	var result struct {
		Status string `json:"status"`
	}
	if len(payload.Result) > 0 && json.Unmarshal(payload.Result, &result) == nil {
		switch strings.TrimSpace(strings.ToLower(result.Status)) {
		case "", "active", "verified":
			status = "healthy"
		default:
			status = result.Status
		}
	}

	return ValidationResult{
		Status: status,
		Capabilities: []string{
			"tokens.verify",
			"zones.read",
			"dns.read",
			"dns.write",
		},
	}, nil
}

func (a *CloudflareAdapter) ListZones(ctx context.Context, account Account) ([]Zone, error) {
	credentials, err := ResolveCloudflareCredentials(account.SecretRef)
	if err != nil {
		return nil, err
	}

	endpoint, err := joinURL(a.baseURL, "/zones")
	if err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, endpoint, map[string]string{
		"page":     "1",
		"per_page": "100",
	})
	if err != nil {
		return nil, err
	}
	if err := applyCloudflareAuthHeaders(req, account.AuthType, credentials); err != nil {
		return nil, err
	}

	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Success bool                 `json:"success"`
		Errors  []cloudflareAPIError `json:"errors"`
		Result  []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"result"`
	}
	if err := decodeJSONResponse(response, &payload); err != nil {
		return nil, err
	}
	if err := ensureCloudflareSuccess(payload.Success, payload.Errors); err != nil {
		return nil, err
	}

	zones := make([]Zone, 0, len(payload.Result))
	for _, item := range payload.Result {
		zones = append(zones, Zone{
			ID:     item.ID,
			Name:   item.Name,
			Status: item.Status,
		})
	}
	return zones, nil
}

func (a *CloudflareAdapter) ListRecords(ctx context.Context, account Account, zoneID string) ([]Record, error) {
	credentials, err := ResolveCloudflareCredentials(account.SecretRef)
	if err != nil {
		return nil, err
	}

	endpoint, err := joinURL(a.baseURL, "/zones/"+zoneID+"/dns_records")
	if err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, endpoint, map[string]string{
		"page":     "1",
		"per_page": "100",
	})
	if err != nil {
		return nil, err
	}
	if err := applyCloudflareAuthHeaders(req, account.AuthType, credentials); err != nil {
		return nil, err
	}

	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Success bool                 `json:"success"`
		Errors  []cloudflareAPIError `json:"errors"`
		Result  []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Name     string `json:"name"`
			Content  string `json:"content"`
			TTL      int    `json:"ttl"`
			Priority int    `json:"priority"`
			Proxied  bool   `json:"proxied"`
		} `json:"result"`
	}
	if err := decodeJSONResponse(response, &payload); err != nil {
		return nil, err
	}
	if err := ensureCloudflareSuccess(payload.Success, payload.Errors); err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(payload.Result))
	for _, item := range payload.Result {
		records = append(records, Record{
			ID:       item.ID,
			Type:     item.Type,
			Name:     item.Name,
			Value:    item.Content,
			TTL:      item.TTL,
			Priority: item.Priority,
			Proxied:  item.Proxied,
		})
	}
	return records, nil
}

func (a *CloudflareAdapter) ApplyChanges(ctx context.Context, account Account, zoneID string, changes []Change) error {
	credentials, err := ResolveCloudflareCredentials(account.SecretRef)
	if err != nil {
		return err
	}

	for _, change := range changes {
		switch change.Operation {
		case "create":
			if change.After == nil {
				continue
			}
			endpoint, err := joinURL(a.baseURL, "/zones/"+zoneID+"/dns_records")
			if err != nil {
				return err
			}
			req, err := newJSONRequest(ctx, http.MethodPost, endpoint, nil, cloudflareRecordPayload(*change.After))
			if err != nil {
				return err
			}
			if err := applyCloudflareAuthHeaders(req, account.AuthType, credentials); err != nil {
				return err
			}
			response, err := a.client.Do(req)
			if err != nil {
				return err
			}
			if err := ensureCloudflareMutationResponse(response); err != nil {
				return err
			}
		case "update":
			if change.Before == nil || change.After == nil || change.Before.ID == "" {
				continue
			}
			endpoint, err := joinURL(a.baseURL, "/zones/"+zoneID+"/dns_records/"+change.Before.ID)
			if err != nil {
				return err
			}
			req, err := newJSONRequest(ctx, http.MethodPatch, endpoint, nil, cloudflareRecordPayload(*change.After))
			if err != nil {
				return err
			}
			if err := applyCloudflareAuthHeaders(req, account.AuthType, credentials); err != nil {
				return err
			}
			response, err := a.client.Do(req)
			if err != nil {
				return err
			}
			if err := ensureCloudflareMutationResponse(response); err != nil {
				return err
			}
		case "delete":
			if change.Before == nil || change.Before.ID == "" {
				continue
			}
			endpoint, err := joinURL(a.baseURL, "/zones/"+zoneID+"/dns_records/"+change.Before.ID)
			if err != nil {
				return err
			}
			req, err := newRequest(ctx, http.MethodDelete, endpoint, nil)
			if err != nil {
				return err
			}
			if err := applyCloudflareAuthHeaders(req, account.AuthType, credentials); err != nil {
				return err
			}
			response, err := a.client.Do(req)
			if err != nil {
				return err
			}
			if err := ensureCloudflareMutationResponse(response); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyCloudflareAuthHeaders(req *http.Request, authType string, credentials CloudflareCredentials) error {
	switch strings.TrimSpace(strings.ToLower(authType)) {
	case "", "api_token":
		if credentials.APIToken != "" {
			req.Header.Set("Authorization", "Bearer "+credentials.APIToken)
			return nil
		}
		if credentials.APIKey != "" && credentials.Email != "" {
			req.Header.Set("X-Auth-Key", credentials.APIKey)
			req.Header.Set("X-Auth-Email", credentials.Email)
			return nil
		}
	case "api_key", "global_api_key":
		if credentials.APIKey != "" && credentials.Email != "" {
			req.Header.Set("X-Auth-Key", credentials.APIKey)
			req.Header.Set("X-Auth-Email", credentials.Email)
			return nil
		}
		if credentials.APIToken != "" {
			req.Header.Set("Authorization", "Bearer "+credentials.APIToken)
			return nil
		}
	}
	return ErrInvalidProviderSecret
}

func ensureCloudflareSuccess(success bool, errors []cloudflareAPIError) error {
	if success {
		return nil
	}
	if len(errors) == 0 {
		return fmt.Errorf("cloudflare request failed")
	}
	parts := make([]string, 0, len(errors))
	for _, item := range errors {
		message := strings.TrimSpace(item.Message)
		if message == "" {
			continue
		}
		if item.Code != 0 {
			parts = append(parts, fmt.Sprintf("%d: %s", item.Code, message))
		} else {
			parts = append(parts, message)
		}
	}
	if len(parts) == 0 {
		return fmt.Errorf("cloudflare request failed")
	}
	return fmt.Errorf("cloudflare request failed: %s", strings.Join(parts, "; "))
}

func ensureCloudflareMutationResponse(response *http.Response) error {
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("provider request failed with status %d: %s", response.StatusCode, summarizeProviderErrorBody(body))
	}
	if response.StatusCode == http.StatusNoContent || len(body) == 0 {
		return nil
	}

	var payload struct {
		Success bool                 `json:"success"`
		Errors  []cloudflareAPIError `json:"errors"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	return ensureCloudflareSuccess(payload.Success, payload.Errors)
}

func cloudflareRecordPayload(record Record) map[string]any {
	payload := map[string]any{
		"type":    record.Type,
		"name":    record.Name,
		"content": record.Value,
		"ttl":     record.TTL,
	}
	if record.Priority > 0 {
		payload["priority"] = record.Priority
	}
	if record.Type == "A" || record.Type == "AAAA" || record.Type == "CNAME" {
		payload["proxied"] = record.Proxied
	}
	return payload
}
