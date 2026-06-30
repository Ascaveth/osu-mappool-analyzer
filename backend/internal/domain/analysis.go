package domain

import "time"

// ScopeType identifies which kind of node in the Tournament aggregate (or
// the standalone Beatmap aggregate) an Analysis or Finding is about.
type ScopeType string

const (
	ScopeTournament ScopeType = "tournament"
	ScopeStage      ScopeType = "stage"
	ScopeCategory   ScopeType = "category"
	ScopeBeatmap    ScopeType = "beatmap"
)

// Scope is a polymorphic reference to the node an Analysis was run
// against, matching the scope_type/scope_id pattern in
// docs/06-domain-model.md's ERD.
type Scope struct {
	Type ScopeType
	ID   string
}

// Severity classifies how significant a Finding is. There is no "error"
// level — Findings describe the pool, they don't represent pipeline
// failures.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Finding is one observation produced by an Analyzer. Severity, Reason,
// and Recommendation are required — a Finding without an explanation of
// why it matters is not a valid Finding (docs/04 Architecture Principle 9,
// docs/06-domain-model.md domain rules). The engine enforces this at
// analyzer-result validation time rather than leaving it to convention.
type Finding struct {
	Severity       Severity
	Description    string
	Reason         string
	Recommendation string
	Metrics        map[string]float64

	// TargetStageID is the ID of the Stage this Finding is specifically
	// about, when the Finding's own Scope is broader than one stage (e.g.
	// a tournament-scope progression Finding describing a change between
	// two stages). Empty when the Finding has no single stage to point
	// to. Consumers should prefer this over parsing stage names out of
	// Description/Recommendation text.
	TargetStageID string
}

// Analysis is the structured, persisted result of running one analyzer
// against one Scope. Analysis is immutable once generated: a re-run
// produces a new Analysis, never a mutation of an old one. SourceHash
// captures the analyzer identity plus the exact input data the analyzer
// saw, so two Analyses with equal SourceHash are guaranteed to be
// reproductions of each other (docs/04 Architecture Principle 6:
// determinism).
type Analysis struct {
	ID           string
	AnalyzerName string
	Scope        Scope
	SourceHash   string
	GeneratedAt  time.Time

	// Score is an optional 0.0-1.0 quality signal for this scope, local to
	// this one analyzer's dimension of concern. nil means this analyzer
	// doesn't produce a composite score (some analyzers only ever produce
	// Findings). Scores are deliberately never combined across analyzers
	// into one "tournament score" — see docs/09-analysis-engine-specification.md.
	Score *float64

	Metrics  map[string]float64
	Findings []Finding
}
