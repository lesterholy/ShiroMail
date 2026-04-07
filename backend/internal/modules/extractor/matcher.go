package extractor

import (
	"errors"
	"regexp"
	"slices"
	"strings"
)

const (
	maxPatternLength      = 512
	maxSearchableBodySize = 128 * 1024
)

var ErrInvalidPattern = errors.New("invalid extractor pattern")
var ErrInvalidRule = errors.New("invalid extractor rule")

func normalizeRule(input UpsertRuleInput, sourceType RuleSourceType, ownerUserID *uint64) (Rule, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Rule{}, ErrInvalidRule
	}
	if len(input.TargetFields) == 0 {
		return Rule{}, ErrInvalidRule
	}
	pattern := strings.TrimSpace(input.Pattern)
	if pattern == "" || len(pattern) > maxPatternLength {
		return Rule{}, ErrInvalidRule
	}
	flags := sanitizeFlags(input.Flags)
	if _, err := compilePattern(pattern, flags); err != nil {
		return Rule{}, ErrInvalidPattern
	}
	resultMode := input.ResultMode
	if resultMode == "" {
		resultMode = ResultModeFirstMatch
	}
	if resultMode != ResultModeFirstMatch && resultMode != ResultModeAllMatches && resultMode != ResultModeCaptureGroup {
		return Rule{}, ErrInvalidRule
	}
	if resultMode == ResultModeCaptureGroup {
		if input.CaptureGroupIndex == nil || *input.CaptureGroupIndex < 0 {
			return Rule{}, ErrInvalidRule
		}
	}
	targetFields := uniqueTargetFields(input.TargetFields)
	sortOrder := input.SortOrder
	if sortOrder <= 0 {
		sortOrder = 100
	}
	return Rule{
		OwnerUserID:       ownerUserID,
		SourceType:        sourceType,
		Name:              name,
		Description:       strings.TrimSpace(input.Description),
		Label:             strings.TrimSpace(input.Label),
		Enabled:           input.Enabled,
		TargetFields:      targetFields,
		Pattern:           pattern,
		Flags:             flags,
		ResultMode:        resultMode,
		CaptureGroupIndex: input.CaptureGroupIndex,
		MailboxIDs:        uniqueUint64s(input.MailboxIDs),
		DomainIDs:         uniqueUint64s(input.DomainIDs),
		SenderContains:    strings.TrimSpace(input.SenderContains),
		SubjectContains:   strings.TrimSpace(input.SubjectContains),
		SortOrder:         sortOrder,
	}, nil
}

func extractMatches(rule Rule, content MessageContent) ([]ExtractionMatch, error) {
	if !rule.Enabled || !ruleMatchesScope(rule, content) {
		return nil, nil
	}
	expr, err := compilePattern(rule.Pattern, rule.Flags)
	if err != nil {
		return nil, ErrInvalidPattern
	}
	items := make([]ExtractionMatch, 0)
	for _, field := range rule.TargetFields {
		needle := fieldValue(field, content)
		if needle == "" {
			continue
		}
		fieldMatches := runFieldExtraction(expr, rule, field, needle)
		items = append(items, fieldMatches...)
		if rule.ResultMode == ResultModeFirstMatch && len(fieldMatches) > 0 {
			break
		}
	}
	return items, nil
}

func runFieldExtraction(expr *regexp.Regexp, rule Rule, field TargetField, needle string) []ExtractionMatch {
	switch rule.ResultMode {
	case ResultModeAllMatches:
		all := expr.FindAllStringSubmatch(needle, 25)
		if len(all) == 0 {
			return nil
		}
		values := make([]string, 0, len(all))
		for _, entry := range all {
			values = append(values, pickCaptureValue(entry, nil))
		}
		return []ExtractionMatch{{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Label:       resolvedLabel(rule),
			SourceType:  string(rule.SourceType),
			SourceField: field,
			Value:       values[0],
			Values:      values,
		}}
	case ResultModeCaptureGroup:
		match := expr.FindStringSubmatch(needle)
		if len(match) == 0 {
			return nil
		}
		value := pickCaptureValue(match, rule.CaptureGroupIndex)
		if value == "" {
			return nil
		}
		captureGroup := rule.CaptureGroupIndex
		return []ExtractionMatch{{
			RuleID:       rule.ID,
			RuleName:     rule.Name,
			Label:        resolvedLabel(rule),
			SourceType:   string(rule.SourceType),
			SourceField:  field,
			Value:        value,
			MatchedText:  match[0],
			CaptureGroup: captureGroup,
		}}
	default:
		match := expr.FindStringSubmatch(needle)
		if len(match) == 0 {
			return nil
		}
		value := pickCaptureValue(match, nil)
		if value == "" {
			return nil
		}
		return []ExtractionMatch{{
			RuleID:      rule.ID,
			RuleName:    rule.Name,
			Label:       resolvedLabel(rule),
			SourceType:  string(rule.SourceType),
			SourceField: field,
			Value:       value,
			MatchedText: match[0],
		}}
	}
}

func compilePattern(pattern string, flags string) (*regexp.Regexp, error) {
	var builder strings.Builder
	if strings.Contains(flags, "i") {
		builder.WriteString("(?i)")
	}
	if strings.Contains(flags, "m") {
		builder.WriteString("(?m)")
	}
	if strings.Contains(flags, "s") {
		builder.WriteString("(?s)")
	}
	expr, err := regexp.Compile(builder.String() + pattern)
	if err != nil {
		return nil, err
	}
	return expr, nil
}

func sanitizeFlags(flags string) string {
	allowed := []rune{'i', 'm', 's'}
	filtered := make([]rune, 0, len(flags))
	for _, flag := range []rune(strings.ToLower(strings.TrimSpace(flags))) {
		if !slices.Contains(allowed, flag) || slices.Contains(filtered, flag) {
			continue
		}
		filtered = append(filtered, flag)
	}
	return string(filtered)
}

func ruleMatchesScope(rule Rule, content MessageContent) bool {
	if len(rule.MailboxIDs) > 0 && !slices.Contains(rule.MailboxIDs, content.MailboxID) {
		return false
	}
	if len(rule.DomainIDs) > 0 && !slices.Contains(rule.DomainIDs, content.DomainID) {
		return false
	}
	if rule.SenderContains != "" && !strings.Contains(strings.ToLower(content.FromAddr), strings.ToLower(rule.SenderContains)) {
		return false
	}
	if rule.SubjectContains != "" && !strings.Contains(strings.ToLower(content.Subject), strings.ToLower(rule.SubjectContains)) {
		return false
	}
	return true
}

func fieldValue(field TargetField, content MessageContent) string {
	switch field {
	case TargetFieldSubject:
		return truncateExtractableText(content.Subject)
	case TargetFieldFromAddr:
		return truncateExtractableText(content.FromAddr)
	case TargetFieldToAddr:
		return truncateExtractableText(content.ToAddr)
	case TargetFieldHTMLText:
		return truncateExtractableText(htmlToText(content.HTMLBody))
	case TargetFieldRawText:
		return truncateExtractableText(content.RawText)
	case TargetFieldTextBody:
		return truncateExtractableText(content.TextBody)
	default:
		return ""
	}
}

func truncateExtractableText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxSearchableBodySize {
		return value
	}
	return value[:maxSearchableBodySize]
}

func htmlToText(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	withoutTags := regexp.MustCompile("(?s)<[^>]+>").ReplaceAllString(value, " ")
	withoutEntities := strings.NewReplacer("&nbsp;", " ", "&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(withoutTags)
	return strings.Join(strings.Fields(withoutEntities), " ")
}

func pickCaptureValue(match []string, captureGroupIndex *int) string {
	if len(match) == 0 {
		return ""
	}
	if captureGroupIndex != nil {
		index := *captureGroupIndex
		if index >= 0 && index < len(match) {
			return strings.TrimSpace(match[index])
		}
		return ""
	}
	if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
		return strings.TrimSpace(match[1])
	}
	return strings.TrimSpace(match[0])
}

func uniqueTargetFields(items []TargetField) []TargetField {
	filtered := make([]TargetField, 0, len(items))
	for _, item := range items {
		if item == "" || slices.Contains(filtered, item) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func uniqueUint64s(items []uint64) []uint64 {
	filtered := make([]uint64, 0, len(items))
	for _, item := range items {
		if item == 0 || slices.Contains(filtered, item) {
			continue
		}
		filtered = append(filtered, item)
	}
	slices.Sort(filtered)
	return filtered
}

func resolvedLabel(rule Rule) string {
	if strings.TrimSpace(rule.Label) != "" {
		return strings.TrimSpace(rule.Label)
	}
	return rule.Name
}
