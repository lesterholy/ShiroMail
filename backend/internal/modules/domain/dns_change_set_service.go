package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	domainprovider "shiro-email/backend/internal/modules/domain/provider"
)

var ErrInvalidDNSChangeSetRequest = errors.New("invalid dns change set request")
var ErrUnsupportedDNSRecordType = errors.New("unsupported dns record type")

type PreviewProviderChangeSetRequest struct {
	ZoneName string           `json:"zoneName" binding:"required"`
	Records  []ProviderRecord `json:"records" binding:"required"`
}

func (s *Service) ListProviderChangeSets(ctx context.Context, providerAccountID uint64, providerZoneID string) ([]DNSChangeSet, error) {
	if _, err := s.repo.GetProviderAccountByID(ctx, providerAccountID); err != nil {
		return nil, err
	}
	return s.repo.ListDNSChangeSets(ctx, providerAccountID, strings.TrimSpace(providerZoneID))
}

func (s *Service) PreviewProviderChangeSet(ctx context.Context, providerAccountID uint64, providerZoneID string, requestedByUserID uint64, req PreviewProviderChangeSetRequest) (DNSChangeSet, error) {
	item, err := s.repo.GetProviderAccountByID(ctx, providerAccountID)
	if err != nil {
		return DNSChangeSet{}, err
	}
	if s.providers == nil {
		return DNSChangeSet{}, ErrProviderAdapterUnavailable
	}

	adapter, ok := s.providers.Get(item.Provider)
	if !ok {
		return DNSChangeSet{}, ErrProviderAdapterUnavailable
	}

	currentRecords, err := adapter.ListRecords(ctx, domainprovider.Account{
		ID:        item.ID,
		Provider:  item.Provider,
		AuthType:  item.AuthType,
		SecretRef: item.SecretRef,
	}, strings.TrimSpace(providerZoneID))
	if err != nil {
		return DNSChangeSet{}, err
	}

	desiredRecords, err := sanitizeDesiredProviderRecords(req.Records)
	if err != nil {
		return DNSChangeSet{}, err
	}

	operations := diffProviderRecords(currentRecords, desiredRecords)
	now := time.Now()
	changeSet := DNSChangeSet{
		ProviderAccountID: providerAccountID,
		ProviderZoneID:    strings.TrimSpace(providerZoneID),
		ZoneName:          strings.TrimSpace(req.ZoneName),
		RequestedByUserID: requestedByUserID,
		Status:            "previewed",
		Provider:          item.Provider,
		Summary:           summarizeDNSChangeOperations(operations),
		Operations:        operations,
		CreatedAt:         now,
	}
	return s.repo.SaveDNSChangeSet(ctx, changeSet)
}

func (s *Service) ApplyProviderChangeSet(ctx context.Context, changeSetID uint64) (DNSChangeSet, error) {
	changeSet, err := s.repo.GetDNSChangeSetByID(ctx, changeSetID)
	if err != nil {
		return DNSChangeSet{}, err
	}
	if changeSet.Status == "applied" {
		return changeSet, nil
	}

	account, err := s.repo.GetProviderAccountByID(ctx, changeSet.ProviderAccountID)
	if err != nil {
		return DNSChangeSet{}, err
	}
	if s.providers == nil {
		return DNSChangeSet{}, ErrProviderAdapterUnavailable
	}

	adapter, ok := s.providers.Get(account.Provider)
	if !ok {
		return DNSChangeSet{}, ErrProviderAdapterUnavailable
	}

	changes := make([]domainprovider.Change, 0, len(changeSet.Operations))
	for _, operation := range changeSet.Operations {
		changes = append(changes, domainprovider.Change{
			Operation: operation.Operation,
			Before:    toProviderAdapterRecord(operation.Before),
			After:     toProviderAdapterRecord(operation.After),
		})
	}

	if err := adapter.ApplyChanges(ctx, domainprovider.Account{
		ID:        account.ID,
		Provider:  account.Provider,
		AuthType:  account.AuthType,
		SecretRef: account.SecretRef,
	}, changeSet.ProviderZoneID, changes); err != nil {
		return DNSChangeSet{}, err
	}

	appliedAt := time.Now()
	changeSet.Status = "applied"
	changeSet.AppliedAt = &appliedAt
	for index := range changeSet.Operations {
		changeSet.Operations[index].Status = "applied"
	}
	return s.repo.SaveDNSChangeSet(ctx, changeSet)
}

func sanitizeDesiredProviderRecords(records []ProviderRecord) ([]ProviderRecord, error) {
	items := make([]ProviderRecord, 0, len(records))
	for _, record := range records {
		normalized, err := normalizeDesiredProviderRecord(record)
		if err != nil {
			return nil, err
		}
		items = append(items, normalized)
	}
	return items, nil
}

func normalizeDesiredProviderRecord(record ProviderRecord) (ProviderRecord, error) {
	normalized := normalizeProviderRecord(record)
	if normalized.Type == "" || normalized.Name == "" || normalized.Value == "" {
		return ProviderRecord{}, ErrInvalidDNSChangeSetRequest
	}
	if _, ok := supportedDNSRecordTypes()[normalized.Type]; !ok {
		return ProviderRecord{}, fmt.Errorf("%w: %s", ErrUnsupportedDNSRecordType, normalized.Type)
	}
	return normalized, nil
}

func normalizeProviderRecord(record ProviderRecord) ProviderRecord {
	record.ID = strings.TrimSpace(record.ID)
	record.Type = strings.ToUpper(strings.TrimSpace(record.Type))
	record.Name = strings.TrimSpace(record.Name)
	record.Value = strings.TrimSpace(record.Value)
	if record.TTL < 0 {
		record.TTL = 0
	}
	if record.Priority < 0 {
		record.Priority = 0
	}
	return record
}

func diffProviderRecords(current []domainprovider.Record, desired []ProviderRecord) []DNSChangeOperation {
	currentItems := make([]ProviderRecord, 0, len(current))
	for _, record := range current {
		currentItems = append(currentItems, normalizeProviderRecord(ProviderRecord{
			ID:       record.ID,
			Type:     record.Type,
			Name:     record.Name,
			Value:    record.Value,
			TTL:      record.TTL,
			Priority: record.Priority,
			Proxied:  record.Proxied,
		}))
	}

	desiredItems := make([]ProviderRecord, 0, len(desired))
	for _, record := range desired {
		desiredItems = append(desiredItems, normalizeProviderRecord(record))
	}

	usedCurrent := make([]bool, len(currentItems))
	usedDesired := make([]bool, len(desiredItems))
	operations := make([]DNSChangeOperation, 0)

	currentGroups := groupProviderRecordsByIdentity(currentItems)
	desiredGroups := groupProviderRecordsByIdentity(desiredItems)

	seenKeys := map[string]struct{}{}
	keys := make([]string, 0, len(currentGroups)+len(desiredGroups))
	for key := range currentGroups {
		keys = append(keys, key)
		seenKeys[key] = struct{}{}
	}
	for key := range desiredGroups {
		if _, ok := seenKeys[key]; ok {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		currentIndexes := currentGroups[key]
		desiredIndexes := desiredGroups[key]
		if len(currentIndexes) != 1 || len(desiredIndexes) != 1 {
			continue
		}

		currentRecord := currentItems[currentIndexes[0]]
		desiredRecord := desiredItems[desiredIndexes[0]]
		usedCurrent[currentIndexes[0]] = true
		usedDesired[desiredIndexes[0]] = true
		if providerRecordsEqual(currentRecord, desiredRecord) {
			continue
		}
		operations = append(operations, DNSChangeOperation{
			Operation:  "update",
			RecordType: desiredRecord.Type,
			RecordName: desiredRecord.Name,
			Before:     cloneProviderRecordPtr(&currentRecord),
			After:      cloneProviderRecordPtr(&desiredRecord),
			Status:     "pending",
		})
	}

	for desiredIndex, desiredRecord := range desiredItems {
		if usedDesired[desiredIndex] {
			continue
		}
		for currentIndex, currentRecord := range currentItems {
			if usedCurrent[currentIndex] {
				continue
			}
			if providerRecordSignature(currentRecord) != providerRecordSignature(desiredRecord) {
				continue
			}
			usedCurrent[currentIndex] = true
			usedDesired[desiredIndex] = true
			break
		}
	}

	for currentIndex, currentRecord := range currentItems {
		if usedCurrent[currentIndex] {
			continue
		}
		operations = append(operations, DNSChangeOperation{
			Operation:  "delete",
			RecordType: currentRecord.Type,
			RecordName: currentRecord.Name,
			Before:     cloneProviderRecordPtr(&currentRecord),
			Status:     "pending",
		})
	}

	for desiredIndex, desiredRecord := range desiredItems {
		if usedDesired[desiredIndex] {
			continue
		}
		operations = append(operations, DNSChangeOperation{
			Operation:  "create",
			RecordType: desiredRecord.Type,
			RecordName: desiredRecord.Name,
			After:      cloneProviderRecordPtr(&desiredRecord),
			Status:     "pending",
		})
	}

	sort.SliceStable(operations, func(i, j int) bool {
		leftRank := dnsChangeOperationRank(operations[i].Operation)
		rightRank := dnsChangeOperationRank(operations[j].Operation)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if operations[i].RecordName != operations[j].RecordName {
			return operations[i].RecordName < operations[j].RecordName
		}
		return operations[i].RecordType < operations[j].RecordType
	})

	return operations
}

func groupProviderRecordsByIdentity(records []ProviderRecord) map[string][]int {
	groups := make(map[string][]int, len(records))
	for index, record := range records {
		key := providerRecordIdentity(record)
		groups[key] = append(groups[key], index)
	}
	return groups
}

func providerRecordsEqual(left ProviderRecord, right ProviderRecord) bool {
	return providerRecordSignature(left) == providerRecordSignature(right)
}

func providerRecordIdentity(record ProviderRecord) string {
	return strings.Join([]string{
		record.Type,
		strings.ToLower(record.Name),
		strconv.Itoa(record.Priority),
	}, "|")
}

func providerRecordSignature(record ProviderRecord) string {
	return strings.Join([]string{
		providerRecordIdentity(record),
		record.Value,
		strconv.Itoa(record.TTL),
		strconv.FormatBool(record.Proxied),
	}, "|")
}

func summarizeDNSChangeOperations(operations []DNSChangeOperation) string {
	createCount := 0
	updateCount := 0
	deleteCount := 0
	for _, operation := range operations {
		switch operation.Operation {
		case "create":
			createCount++
		case "update":
			updateCount++
		case "delete":
			deleteCount++
		}
	}
	return fmt.Sprintf("%d create, %d update, %d delete", createCount, updateCount, deleteCount)
}

func dnsChangeOperationRank(operation string) int {
	switch operation {
	case "create":
		return 0
	case "update":
		return 1
	case "delete":
		return 2
	default:
		return 3
	}
}

func supportedDNSRecordTypes() map[string]struct{} {
	return map[string]struct{}{
		"A":     {},
		"AAAA":  {},
		"CNAME": {},
		"TXT":   {},
		"MX":    {},
		"NS":    {},
	}
}

func toProviderAdapterRecord(record *ProviderRecord) *domainprovider.Record {
	if record == nil {
		return nil
	}
	return &domainprovider.Record{
		ID:       record.ID,
		Type:     record.Type,
		Name:     record.Name,
		Value:    record.Value,
		TTL:      record.TTL,
		Priority: record.Priority,
		Proxied:  record.Proxied,
	}
}

func cloneProviderRecordPtr(record *ProviderRecord) *ProviderRecord {
	if record == nil {
		return nil
	}
	cloned := *record
	return &cloned
}
