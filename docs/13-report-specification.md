# Report Specification

Phase 9 deliverable: turning `domain.Analysis`/`domain.Finding` output into a human-readable `domain.Report`. Implemented in `backend/internal/report` (builder) and `backend/internal/domain/report.go` (types).

## What a Report is

A Report is a **derived view**, exactly like an Analysis — it narrates Analyses already produced by the Analysis Engine, it never introduces a new conclusion of its own (`docs/06-domain-model.md`: *"a Report must not contain data that doesn't trace back to at least one cited Analysis"*). `report.Build` is a pure function: given the same Analyses, it always produces the same Report, and it performs no analysis of its own — only selection, counting, and arrangement.

```go
func Build(scope domain.Scope, analyses []domain.Analysis, now func() time.Time) domain.Report
```

The caller decides which Analyses go into a Report — e.g. every Analysis from one `Engine.Run` for a whole-tournament report, or a pre-filtered subtree for a narrower one. `Build` has no knowledge of the Tournament tree; this keeps it testable against synthetic data and reusable at any scope, and keeps tree-traversal logic (already owned by `internal/analysis`) out of a package whose only job is narration.

## Sections

Matches the roadmap's Phase 9 outputs exactly: Summary, Findings, Warnings, Recommendations, Statistics.

| Section | Content | Source |
|---|---|---|
| **Summary** | Prose: what happened, why it matters | `Finding.Description` text, counted and concatenated — never raw metric values |
| **Findings** | Every Finding, cited back to its Analyzer and Scope | `domain.Citation{AnalyzerName, Scope, Finding}` |
| **Warnings** | Findings with severity `warning` or `critical` | Same Citations, filtered |
| **Recommendations** | Deduplicated `Finding.Recommendation` strings, first-appearance order | Citations |
| **Statistics** | Counts only: total analyses, total findings, findings by severity, analyses with a score, average score | Computed from Analyses/Citations |

### Why Summary and Statistics are separate sections

Architecture Principle 9 (*"Reports speak in conclusions, not raw numbers"*) constrains the **Summary** specifically — a summary that says "average BPM increased by 12" fails the principle, while "the Finals stage introduces a noticeable increase in technical difficulty" passes it. It does not forbid raw numbers from existing anywhere in a Report; it forbids a *conclusion* from being reduced to *only* a number. Roadmap Phase 9 names Statistics as its own output alongside Summary, so `Statistics` holds the counts, and `Summary` holds prose built from Findings' own `Description` text — which every analyzer already writes in conclusion form (e.g. `progression.go`'s spike finding: *"average OD increases by 6.50 from 'RO16' to 'Finals', more than 2x the tournament's typical stage-to-stage increase"*). The Summary never restates a Statistics number on its own; it only counts and quotes existing Finding prose.

### Findings are ranked, not just listed

Within a Report, `Findings` (and `Warnings`) are sorted critical-first, then warning, then info, so a reader sees what matters most without scanning the whole list. Ties within a severity tier preserve the Engine's own deterministic order (analyzer name, then scope ID) rather than introducing a new ordering rule.

## Why citation is by (AnalyzerName, Scope), not Analysis ID

`docs/06-domain-model.md`'s ERD gives `Analysis` a `uuid id PK`, and a Report should cite Analyses by that ID. As of Phase 9, no `AnalysisRepository` exists — `Engine.Run` never persists or assigns IDs to the Analyses it produces (only `Beatmap` has a repository so far, see `internal/storage/memory`). Until Phase 10 introduces analysis persistence, `domain.Citation` uses `(AnalyzerName, Scope)` as its key, which already uniquely identifies one Analysis within a single `Engine.Run` — it's the same pair `Engine.Run` itself sorts results by. Swapping this for a persisted `Analysis.ID` later is a one-field change to `Citation`, not a redesign.

## What is explicitly not in scope for this phase

- **Report persistence.** `domain.Report` is a value type returned by `Build`; there is no `ReportRepository`. Reports are cheap to regenerate from Analyses (consistent with Architecture Principle 5: derived data is never duplicated), so persisting them is deferred until there's an actual need (e.g. serving a stored Report via the Phase 10 API without recomputing it).
- **Output formatting (HTML/Markdown/PDF rendering).** `domain.Report` is a structured Go value. Rendering it to a specific output format is a presentation concern for Phase 10 (REST API, as JSON) and Phase 11 (frontend), not the Analysis-Engine-adjacent reporting layer built here.
- **Cross-tournament / historical reports.** Comparing one tournament's Report to another's is listed as a Future Idea (`pool-lab-plan.md`); `Build` operates on one set of Analyses from one Tournament at a time.

## Testing

`backend/internal/report/report_test.go`, 4 tests:

- `TestBuild_NoFindingsProducesPositiveSummary` — zero Findings still produces a non-empty, positively-framed Summary rather than an empty string.
- `TestBuild_FindingsAreCitedAndCounted` — Findings are correctly cited, severity-ranked (critical before warning), counted in Statistics, and a non-nil Score is correctly averaged.
- `TestBuild_DeduplicatesRecommendations` — two different Analyzers independently producing the same Recommendation text collapse to one entry.
- `TestBuild_SetsScopeAndTimestamp` — Report identity fields (`ScopeType`, `ScopeID`, `GeneratedAt`) are set correctly, including the zero-Analyses case.

Run with:

```sh
cd backend && go test ./...
```
