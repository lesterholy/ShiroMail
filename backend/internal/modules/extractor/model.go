package extractor

import "time"

type RuleSourceType string

const (
	RuleSourceUser         RuleSourceType = "user"
	RuleSourceAdminDefault RuleSourceType = "admin_default"
)

type TargetField string

const (
	TargetFieldSubject  TargetField = "subject"
	TargetFieldFromAddr TargetField = "from_addr"
	TargetFieldToAddr   TargetField = "to_addr"
	TargetFieldTextBody TargetField = "text_body"
	TargetFieldHTMLText TargetField = "html_text"
	TargetFieldRawText  TargetField = "raw_text"
)

type ResultMode string

const (
	ResultModeFirstMatch   ResultMode = "first_match"
	ResultModeAllMatches   ResultMode = "all_matches"
	ResultModeCaptureGroup ResultMode = "capture_group"
)

type Rule struct {
	ID                uint64         `json:"id"`
	OwnerUserID       *uint64        `json:"ownerUserId,omitempty"`
	SourceType        RuleSourceType `json:"sourceType"`
	TemplateKey       string         `json:"templateKey,omitempty"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Label             string         `json:"label"`
	Enabled           bool           `json:"enabled"`
	TargetFields      []TargetField  `json:"targetFields"`
	Pattern           string         `json:"pattern"`
	Flags             string         `json:"flags"`
	ResultMode        ResultMode     `json:"resultMode"`
	CaptureGroupIndex *int           `json:"captureGroupIndex,omitempty"`
	MailboxIDs        []uint64       `json:"mailboxIds,omitempty"`
	DomainIDs         []uint64       `json:"domainIds,omitempty"`
	SenderContains    string         `json:"senderContains,omitempty"`
	SubjectContains   string         `json:"subjectContains,omitempty"`
	SortOrder         int            `json:"sortOrder"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type RuleList struct {
	Rules     []RuleView `json:"rules"`
	Templates []RuleView `json:"templates"`
}

type RuleView struct {
	Rule
	EnabledForUser bool `json:"enabledForUser"`
}

type MessageContent struct {
	MailboxID uint64 `json:"mailboxId"`
	DomainID  uint64 `json:"domainId"`

	Subject  string `json:"subject"`
	FromAddr string `json:"fromAddr"`
	ToAddr   string `json:"toAddr"`
	TextBody string `json:"textBody"`
	HTMLBody string `json:"htmlBody"`
	RawText  string `json:"rawText"`
}

type ExtractionMatch struct {
	RuleID       uint64      `json:"ruleId"`
	RuleName     string      `json:"ruleName"`
	Label        string      `json:"label"`
	SourceType   string      `json:"sourceType"`
	SourceField  TargetField `json:"sourceField"`
	Value        string      `json:"value"`
	Values       []string    `json:"values,omitempty"`
	MatchedText  string      `json:"matchedText,omitempty"`
	CaptureGroup *int        `json:"captureGroup,omitempty"`
}

type ExtractionResult struct {
	Items []ExtractionMatch `json:"items"`
}

type RuleTestResult struct {
	Items []ExtractionMatch `json:"items"`
}

type UpsertRuleInput struct {
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	Label             string        `json:"label"`
	Enabled           bool          `json:"enabled"`
	TargetFields      []TargetField `json:"targetFields"`
	Pattern           string        `json:"pattern"`
	Flags             string        `json:"flags"`
	ResultMode        ResultMode    `json:"resultMode"`
	CaptureGroupIndex *int          `json:"captureGroupIndex,omitempty"`
	MailboxIDs        []uint64      `json:"mailboxIds"`
	DomainIDs         []uint64      `json:"domainIds"`
	SenderContains    string        `json:"senderContains"`
	SubjectContains   string        `json:"subjectContains"`
	SortOrder         int           `json:"sortOrder"`
}

type RuleTestInput struct {
	MailboxID *uint64 `json:"mailboxId,omitempty"`
	MessageID *uint64 `json:"messageId,omitempty"`
	Subject   string  `json:"subject,omitempty"`
	FromAddr  string  `json:"fromAddr,omitempty"`
	ToAddr    string  `json:"toAddr,omitempty"`
	TextBody  string  `json:"textBody,omitempty"`
	HTMLBody  string  `json:"htmlBody,omitempty"`
	RawText   string  `json:"rawText,omitempty"`
}
