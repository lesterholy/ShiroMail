package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const defaultSpaceshipBaseURL = "https://spaceship.dev/api"

type SpaceshipAdapter struct {
	baseURL string
	client  *http.Client
}

func NewSpaceshipAdapter(baseURL string, client *http.Client) *SpaceshipAdapter {
	if baseURL == "" {
		baseURL = defaultSpaceshipBaseURL
	}
	return &SpaceshipAdapter{
		baseURL: baseURL,
		client:  defaultHTTPClient(client),
	}
}

func (a *SpaceshipAdapter) Validate(ctx context.Context, account Account) (ValidationResult, error) {
	credentials, err := ResolveSpaceshipCredentials(account.SecretRef)
	if err != nil {
		return ValidationResult{}, err
	}

	endpoint, err := joinURL(a.baseURL, "/v1/domains")
	if err != nil {
		return ValidationResult{}, err
	}

	req, err := newRequest(ctx, http.MethodGet, endpoint, map[string]string{
		"take": "1",
		"skip": "0",
	})
	if err != nil {
		return ValidationResult{}, err
	}
	req.Header.Set("X-API-Key", credentials.APIKey)
	req.Header.Set("X-API-Secret", credentials.APISecret)

	response, err := a.client.Do(req)
	if err != nil {
		return ValidationResult{}, err
	}

	var payload struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := decodeJSONResponse(response, &payload); err != nil {
		return ValidationResult{}, err
	}
	return ValidationResult{
		Status: "healthy",
		Capabilities: []string{
			"zones.read",
			"dns.read",
			"dns.write",
		},
	}, nil
}

func (a *SpaceshipAdapter) ListZones(ctx context.Context, account Account) ([]Zone, error) {
	credentials, err := ResolveSpaceshipCredentials(account.SecretRef)
	if err != nil {
		return nil, err
	}

	endpoint, err := joinURL(a.baseURL, "/v1/domains")
	if err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, endpoint, map[string]string{
		"take": "100",
		"skip": "0",
	})
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", credentials.APIKey)
	req.Header.Set("X-API-Secret", credentials.APISecret)

	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Items []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := decodeJSONResponse(response, &payload); err != nil {
		return nil, err
	}

	zones := make([]Zone, 0, len(payload.Items))
	for _, item := range payload.Items {
		zones = append(zones, Zone{
			ID:     item.Name,
			Name:   item.Name,
			Status: item.Status,
		})
	}
	return zones, nil
}

func (a *SpaceshipAdapter) ListRecords(ctx context.Context, account Account, zoneName string) ([]Record, error) {
	credentials, err := ResolveSpaceshipCredentials(account.SecretRef)
	if err != nil {
		return nil, err
	}

	escapedZone := url.PathEscape(zoneName)
	endpoint, err := joinURL(a.baseURL, "/v1/dns/records/"+escapedZone)
	if err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, endpoint, map[string]string{
		"take": "100",
		"skip": "0",
	})
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", credentials.APIKey)
	req.Header.Set("X-API-Secret", credentials.APISecret)

	response, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Items []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Name       string `json:"name"`
			Value      string `json:"value"`
			Address    string `json:"address"`
			CNAME      string `json:"cname"`
			Exchange   string `json:"exchange"`
			Target     string `json:"target"`
			TTL        int    `json:"ttl"`
			Priority   int    `json:"priority"`
			Preference int    `json:"preference"`
		} `json:"items"`
	}
	if err := decodeJSONResponse(response, &payload); err != nil {
		return nil, err
	}

	items := make([]Record, 0, len(payload.Items))
	for _, item := range payload.Items {
		record := Record{
			ID:       item.ID,
			Type:     strings.ToUpper(strings.TrimSpace(item.Type)),
			Name:     spaceshipRecordNameToFQDN(zoneName, item.Name),
			Value:    spaceshipRecordResponseValue(item.Type, item.Value, item.Address, item.CNAME, item.Exchange, item.Target),
			TTL:      item.TTL,
			Priority: spaceshipRecordResponsePriority(item.Priority, item.Preference),
		}
		if record.ID == "" {
			record.ID = fmt.Sprintf("%s:%s:%s", record.Type, record.Name, record.Value)
		}
		items = append(items, record)
	}
	return items, nil
}

func (a *SpaceshipAdapter) ApplyChanges(ctx context.Context, account Account, zoneName string, changes []Change) error {
	credentials, err := ResolveSpaceshipCredentials(account.SecretRef)
	if err != nil {
		return err
	}

	escapedZone := url.PathEscape(zoneName)
	endpoint, err := joinURL(a.baseURL, "/v1/dns/records/"+escapedZone)
	if err != nil {
		return err
	}

	saveItems := make([]map[string]any, 0, len(changes))
	deleteItems := make([]map[string]any, 0, len(changes))
	for _, change := range changes {
		switch change.Operation {
		case "create", "update":
			if change.After == nil {
				continue
			}
			saveItems = append(saveItems, spaceshipRecordPayload(zoneName, *change.After))
		case "delete":
			if change.Before == nil {
				continue
			}
			deleteItems = append(deleteItems, spaceshipRecordPayload(zoneName, *change.Before))
		}
	}

	if len(saveItems) > 0 {
		req, err := newJSONRequest(ctx, http.MethodPut, endpoint, nil, map[string]any{
			"force": false,
			"items": saveItems,
		})
		if err != nil {
			return err
		}
		req.Header.Set("X-API-Key", credentials.APIKey)
		req.Header.Set("X-API-Secret", credentials.APISecret)

		response, err := a.client.Do(req)
		if err != nil {
			return err
		}
		if err := ensureSuccessResponse(response); err != nil {
			return err
		}
	}

	if len(deleteItems) > 0 {
		req, err := newJSONRequest(ctx, http.MethodDelete, endpoint, nil, deleteItems)
		if err != nil {
			return err
		}
		req.Header.Set("X-API-Key", credentials.APIKey)
		req.Header.Set("X-API-Secret", credentials.APISecret)

		response, err := a.client.Do(req)
		if err != nil {
			return err
		}
		if err := ensureSuccessResponse(response); err != nil {
			return err
		}
	}

	return nil
}

func spaceshipRecordPayload(zoneName string, record Record) map[string]any {
	payload := map[string]any{
		"type": record.Type,
		"name": spaceshipRecordNameToRelative(zoneName, record.Name),
	}
	if record.TTL > 0 {
		payload["ttl"] = record.TTL
	}
	switch record.Type {
	case "A", "AAAA":
		payload["address"] = record.Value
	case "CNAME":
		payload["cname"] = record.Value
	case "MX":
		payload["exchange"] = record.Value
		if record.Priority > 0 {
			payload["preference"] = record.Priority
		}
	default:
		payload["value"] = record.Value
	}
	return payload
}

func spaceshipRecordNameToFQDN(zoneName string, recordName string) string {
	zone := normalizeDomainName(zoneName)
	name := normalizeDomainName(recordName)

	switch {
	case zone == "":
		return name
	case name == "", name == "@":
		return zone
	case name == zone, strings.HasSuffix(name, "."+zone):
		return name
	default:
		return name + "." + zone
	}
}

func spaceshipRecordNameToRelative(zoneName string, recordName string) string {
	zone := normalizeDomainName(zoneName)
	name := normalizeDomainName(recordName)

	switch {
	case zone == "":
		if name == "" {
			return "@"
		}
		return name
	case name == "", name == "@", name == zone:
		return "@"
	case strings.HasSuffix(name, "."+zone):
		host := strings.TrimSuffix(name, "."+zone)
		if host == "" {
			return "@"
		}
		return host
	default:
		return name
	}
}

func normalizeDomainName(value string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(value)), ".")
}

func spaceshipRecordResponseValue(recordType string, value string, address string, cname string, exchange string, target string) string {
	switch strings.ToUpper(strings.TrimSpace(recordType)) {
	case "A", "AAAA":
		if strings.TrimSpace(address) != "" {
			return strings.TrimSpace(address)
		}
	case "CNAME":
		if strings.TrimSpace(cname) != "" {
			return strings.TrimSpace(cname)
		}
	case "MX":
		if strings.TrimSpace(exchange) != "" {
			return strings.TrimSpace(exchange)
		}
	}

	if strings.TrimSpace(target) != "" {
		return strings.TrimSpace(target)
	}
	return strings.TrimSpace(value)
}

func spaceshipRecordResponsePriority(priority int, preference int) int {
	if priority > 0 {
		return priority
	}
	return preference
}
