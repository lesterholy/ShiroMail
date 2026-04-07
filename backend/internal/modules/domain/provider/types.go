package provider

import "context"

type Account struct {
	ID        uint64
	Provider  string
	AuthType  string
	SecretRef string
}

type ValidationResult struct {
	Status       string
	Capabilities []string
}

type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Record struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Proxied  bool   `json:"proxied"`
}

type Change struct {
	Operation string  `json:"operation"`
	Before    *Record `json:"before,omitempty"`
	After     *Record `json:"after,omitempty"`
}

type Adapter interface {
	Validate(ctx context.Context, account Account) (ValidationResult, error)
	ListZones(ctx context.Context, account Account) ([]Zone, error)
	ListRecords(ctx context.Context, account Account, zoneID string) ([]Record, error)
	ApplyChanges(ctx context.Context, account Account, zoneID string, changes []Change) error
}
