package domain

import "time"

type Domain struct {
	ID                  uint64  `json:"id"`
	Domain              string  `json:"domain"`
	Status              string  `json:"status"`
	OwnerUserID         *uint64 `json:"ownerUserId,omitempty"`
	Visibility          string  `json:"visibility"`
	PublicationStatus   string  `json:"publicationStatus"`
	VerificationScore   int     `json:"verificationScore"`
	HealthStatus        string  `json:"healthStatus"`
	ProviderAccountID   *uint64 `json:"providerAccountId,omitempty"`
	Provider            string  `json:"provider,omitempty"`
	ProviderDisplayName string  `json:"providerDisplayName,omitempty"`
	IsDefault           bool    `json:"isDefault"`
	Weight              int     `json:"weight"`
	RootDomain          string  `json:"rootDomain"`
	ParentDomain        string  `json:"parentDomain"`
	Level               int     `json:"level"`
	Kind                string  `json:"kind"`
}

type ProviderAccount struct {
	ID           uint64     `json:"id"`
	Provider     string     `json:"provider"`
	OwnerType    string     `json:"ownerType"`
	OwnerUserID  *uint64    `json:"ownerUserId,omitempty"`
	DisplayName  string     `json:"displayName"`
	AuthType     string     `json:"authType"`
	SecretRef    string     `json:"-"`
	HasSecret    bool       `json:"hasSecret"`
	Status       string     `json:"status"`
	Capabilities []string   `json:"capabilities"`
	LastSyncAt   *time.Time `json:"lastSyncAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type ProviderZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ProviderRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Proxied  bool   `json:"proxied"`
}

type DNSChangeSet struct {
	ID                  uint64               `json:"id"`
	ZoneID              *uint64              `json:"dnsZoneId,omitempty"`
	ProviderAccountID   uint64               `json:"providerAccountId"`
	ProviderZoneID      string               `json:"providerZoneId"`
	ZoneName            string               `json:"zoneName"`
	RequestedByUserID   uint64               `json:"requestedByUserId"`
	RequestedByAPIKeyID *uint64              `json:"requestedByApiKeyId,omitempty"`
	Status              string               `json:"status"`
	Provider            string               `json:"provider"`
	Summary             string               `json:"summary"`
	Operations          []DNSChangeOperation `json:"operations"`
	CreatedAt           time.Time            `json:"createdAt"`
	AppliedAt           *time.Time           `json:"appliedAt,omitempty"`
}

type DNSChangeOperation struct {
	ID          uint64          `json:"id"`
	ChangeSetID uint64          `json:"changeSetId"`
	Operation   string          `json:"operation"`
	RecordType  string          `json:"recordType"`
	RecordName  string          `json:"recordName"`
	Before      *ProviderRecord `json:"before,omitempty"`
	After       *ProviderRecord `json:"after,omitempty"`
	Status      string          `json:"status"`
}

type VerificationProfile struct {
	VerificationType string           `json:"verificationType"`
	Status           string           `json:"status"`
	Summary          string           `json:"summary"`
	ExpectedRecords  []ProviderRecord `json:"expectedRecords"`
	ObservedRecords  []ProviderRecord `json:"observedRecords"`
	RepairRecords    []ProviderRecord `json:"repairRecords"`
	LastCheckedAt    time.Time        `json:"lastCheckedAt"`
}
