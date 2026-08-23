package domain

import "time"

type Decision string

const (
	Prohibited    Decision = "prohibited"
	Conditional   Decision = "conditional"
	Allowed       Decision = "allowed"
	Indeterminate Decision = "indeterminate"
)

type Fact struct {
	Path   string `json:"path"`
	Value  any    `json:"value"`
	Source string `json:"source"`
}
type Conversion struct {
	Field   string `json:"field"`
	Input   string `json:"input"`
	Output  string `json:"output"`
	Formula string `json:"formula"`
}
type RuleHit struct {
	RuleID      string   `json:"rule_id"`
	Priority    int      `json:"priority"`
	Outcome     Decision `json:"outcome"`
	Explanation string   `json:"explanation"`
}
type Conflict struct {
	LeftItem  string   `json:"left_item"`
	RightItem string   `json:"right_item,omitempty"`
	Path      []string `json:"path"`
	Message   string   `json:"message"`
}
type Result struct {
	ID              string       `json:"id"`
	ManifestID      string       `json:"manifest_id"`
	Revision        int64        `json:"revision"`
	RulebookVersion string       `json:"rulebook_version"`
	InputDigest     string       `json:"input_digest"`
	Decision        Decision     `json:"decision"`
	Facts           []Fact       `json:"facts"`
	Conversions     []Conversion `json:"conversions"`
	RuleHits        []RuleHit    `json:"rule_hits"`
	Conflicts       []Conflict   `json:"conflicts"`
	Missing         []string     `json:"missing,omitempty"`
	EvaluatedAt     time.Time    `json:"evaluated_at"`
}
