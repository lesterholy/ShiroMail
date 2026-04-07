package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultRequestTimeout = 15 * time.Second

func defaultHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: defaultRequestTimeout}
}

func joinURL(baseURL string, path string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("empty provider base url")
	}
	target, err := url.Parse(base + "/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return "", err
	}
	return target.String(), nil
}

func newRequest(ctx context.Context, method string, rawURL string, query map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	params := req.URL.Query()
	for key, value := range query {
		params.Set(key, value)
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func newJSONRequest(ctx context.Context, method string, rawURL string, query map[string]string, body any) (*http.Request, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	params := req.URL.Query()
	for key, value := range query {
		params.Set(key, value)
	}
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func decodeJSONResponse[T any](response *http.Response, payload *T) error {
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("provider request failed with status %d: %s", response.StatusCode, summarizeProviderErrorBody(body))
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, payload)
}

func ensureSuccessResponse(response *http.Response) error {
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("provider request failed with status %d: %s", response.StatusCode, summarizeProviderErrorBody(body))
	}
	return nil
}

func summarizeProviderErrorBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "empty response body"
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return truncateProviderError(trimmed)
	}

	if errorsValue, ok := payload["errors"].([]any); ok && len(errorsValue) > 0 {
		parts := make([]string, 0, len(errorsValue))
		for _, item := range errorsValue {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			message := strings.TrimSpace(stringValue(entry["message"]))
			if message == "" {
				message = strings.TrimSpace(stringValue(entry["detail"]))
			}
			code := strings.TrimSpace(stringValue(entry["code"]))
			if code != "" && message != "" {
				parts = append(parts, code+": "+message)
				continue
			}
			if message != "" {
				parts = append(parts, message)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}

	for _, key := range []string{"message", "detail", "error", "title"} {
		if value := strings.TrimSpace(stringValue(payload[key])); value != "" {
			return value
		}
	}

	return truncateProviderError(trimmed)
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	default:
		return ""
	}
}

func truncateProviderError(value string) string {
	if len(value) <= 240 {
		return value
	}
	return value[:240] + "..."
}
