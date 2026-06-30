package domain

import "time"

// Citation ties one Finding back to the Analysis that produced it. A
// Report never duplicates Finding data without attribution
// (docs/06-domain-model.md domain rule: "a Report must not contain data
// that doesn't trace back to at least one cited Analysis"). Analyses are
// not yet persisted with their own IDs — no AnalysisRepository exists as
// of Phase 9 — so AnalyzerName plus Scope is used as the citation key,
// since that pair already uniquely identifies one Analysis within a
// single Engine.Run (see Engine.Run's own sort key). This can be swapped
// for a persisted Analysis ID once Phase 10 introduces one, without
// changing Report's shape.
type Citation struct {
	AnalyzerName string
	Scope        Scope
	Finding      Finding
}

// ReportSections is the narrative body of a Report, matching the
// Outputs list in the project roadmap's Phase 9 (Summary, Findings,
// Warnings, Recommendations, Statistics). Every field here is derived
// from, and traceable to, the Citations the Report was built from —
// Build (internal/report) never introduces a claim that isn't grounded
// in an existing Finding.
type ReportSections struct {
	// Summary is prose: what happened and why it matters, per
	// docs/04-architecture-principles.md Principle 9. It is composed from
	// Finding.Description text already written by analyzers, not raw
	// metric values.
	Summary string

	// Findings is every Finding in the report, in deterministic order,
	// each attributed to the Analyzer and Scope that produced it.
	Findings []Citation

	// Warnings is the subset of Findings with severity Warning or
	// Critical — the same data as Findings, filtered, kept separate so a
	// reader can see "what needs attention" without filtering themselves.
	Warnings []Citation

	// Recommendations is the deduplicated list of Finding.Recommendation
	// strings, in order of first appearance.
	Recommendations []string

	// Statistics are the only raw numbers in a Report. They summarize
	// the Citations, e.g. counts by severity — Architecture Principle 9
	// constrains the Summary, not a dedicated Statistics section.
	Statistics map[string]float64
}

// Report is a human-readable document assembled from one or more
// Analyses for a given scope (docs/06-domain-model.md, Report aggregate).
// Like Analysis, a Report is a regenerable derived view: re-running Build
// against the same Analyses always produces the same Report, and a
// Report is never the source of new conclusions, only a narration of
// Analysis output already on hand.
type Report struct {
	ScopeType   ScopeType
	ScopeID     string
	GeneratedAt time.Time
	Sections    ReportSections
}
