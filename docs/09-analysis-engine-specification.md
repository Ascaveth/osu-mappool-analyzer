# Analysis Engine Specification

Phase 5 deliverable. The Analysis Engine is the heart of the system. This document defines the analyzer interface, the run pipeline, the result format, the scoring model, and the recommendation model. Implemented in `backend/internal/analysis` and `backend/internal/domain`.

## Analyzer interface

```go
type Analyzer interface {
    Name() string
    ScopeType() domain.ScopeType
    Analyze(ctx context.Context, input Input) (Result, error)
}
```

- **`Name()`** uniquely identifies the analyzer (e.g. `"composition-analyzer"`). It is part of an `Analysis`'s identity and feeds `SourceHash`, so renaming an analyzer is effectively introducing a new one as far as historical Analyses are concerned.
- **`ScopeType()`** declares which kind of node the analyzer operates on: `tournament`, `stage`, `category`, or `beatmap`. The Engine uses this to decide how many times to run the analyzer and against what.
- **`Analyze`** receives the full `Tournament` aggregate plus the specific `Scope` it's responsible for this run, and returns a `Result` or an error.

Analyzers receive the **whole** tournament, not just their scope's subtree, because some findings are inherently relational — a stage-scoped progression analyzer needs to see neighboring stages to detect a difficulty spike between them. This is a deliberate exception to "scope = visibility"; what's enforced instead is **independence**: analyzers must never call or depend on another analyzer, and the Engine's registry has no mechanism for one analyzer to look up another. Each analyzer is independently unit-testable against synthetic `Input` data, per [Architecture Principle 11](04-architecture-principles.md#11-testability-is-a-design-constraint-not-an-afterthought).

## Pipeline (Engine.Run)

```
Tournament aggregate
       │
       ▼
for each registered Analyzer:
    enumerate Scopes of Analyzer.ScopeType() within the Tournament
    for each Scope:
        result, err := Analyzer.Analyze(ctx, Input{Tournament, Scope})
        validate result.Findings (Severity/Reason/Recommendation required)
        wrap into domain.Analysis{ SourceHash, GeneratedAt, ... }
       │
       ▼
[]domain.Analysis  (+ joined error covering any per-analyzer/scope failures)
```

Key behaviors:

- **Scope enumeration is automatic.** A `ScopeStage` analyzer runs once per `Stage` in the tournament; a `ScopeBeatmap` analyzer runs once per *distinct* beatmap referenced (deduplicated by ID — the same beatmap reused across multiple slots/stages is analyzed once, not once per occurrence).
- **One analyzer's failure never blocks another.** `Run` continues through every analyzer/scope pair regardless of earlier failures, and returns all per-pair errors joined via `errors.Join` alongside whatever `Analysis` results did succeed. A defect in one plugin must not silently hide the results of the others — this is the direct consequence of analyzers being architecturally independent.
- **Result validation is enforced at the engine boundary, not by convention.** Every `Finding` must have a `Severity`, `Reason`, and `Recommendation` or the engine rejects that analyzer/scope result with an error (see `validateFindings` in `engine.go`). This makes the domain rule from [06-domain-model.md](06-domain-model.md#domain-rules) ("a Finding without an explanation of why it matters is not a valid Finding") a compile-adjacent guarantee instead of a hope.
- **Adding a new analyzer never requires touching the Engine.** `Engine.Register` takes any value satisfying the `Analyzer` interface — see `backend/internal/analysis/engine_test.go`, where three demo analyzers (`stageCoverageAnalyzer`, `beatmapPingAnalyzer`, `brokenAnalyzer`) are registered and run without any change to `engine.go`, `scope.go`, or `sourcehash.go`. This is the plugin architecture working in practice, not just in principle.

## Result format

```go
type Result struct {
    Score    *float64          // optional, 0.0-1.0
    Metrics  map[string]float64
    Findings []domain.Finding
}
```

The Engine wraps each `Result` into a `domain.Analysis`:

```go
type Analysis struct {
    ID, AnalyzerName string
    Scope             Scope        // {Type, ID}
    SourceHash        string
    GeneratedAt       time.Time
    Score             *float64
    Metrics           map[string]float64
    Findings          []Finding
}
```

`Finding` (already defined in the Phase 2 domain model) carries `Severity`, `Description`, `Reason`, `Recommendation`, and optional per-finding `Metrics`.

### SourceHash and determinism

`SourceHash` is a sha256 of the analyzer's name plus a canonical, order-independent serialization of exactly the data within the scoped subtree (tournament/stage/category config plus referenced beatmaps' content hashes — see `sourcehash.go`). Two `Analysis` records with equal `SourceHash` are guaranteed reproductions of each other. This is what makes the [Tournament Configuration spec](07-tournament-configuration.md#updating-a-configuration-after-creation)'s promise concrete: editing a stage's slots changes that stage's `SourceHash` (and its ancestors'/descendants' as applicable) without touching unrelated scopes' hashes — verified directly by `TestEngine_DeterministicSourceHash`, which asserts that changing one stage's beatmap assignment changes that stage's hash but leaves a sibling stage's hash untouched.

## Scoring model

Each analyzer may optionally produce a `Score` in `[0.0, 1.0]` — a single quality signal local to that analyzer's one concern (e.g. a coverage analyzer's score is "fraction of slots filled"). Two rules govern scoring:

1. **A score is local to its analyzer and its scope.** It is never combined across analyzers into one aggregate "tournament score." A composition score, a progression score, and a diversity score answer different questions; averaging them would produce a number with no defensible meaning, which conflicts directly with [Architecture Principle 9](04-architecture-principles.md#9-reports-speak-in-conclusions-not-raw-numbers) (every number must carry meaning, not just exist). If a future report needs a single headline number, that's a Reporting-phase (Phase 9) presentation decision made explicitly with stated weighting and caveats — not something the Engine manufactures implicitly.
2. **A score is optional.** Many analyzers (especially detection-oriented ones like a validation analyzer) only ever produce `Findings` and have no sensible single number to report — `Score` is a `*float64` specifically so "not applicable" is representable, not faked as `0`.

## Recommendation model

There is no separate `Recommendation` entity — a recommendation is a required string field on `Finding`, set by the analyzer that raised the finding. This was decided in Phase 2 ([06-domain-model.md](06-domain-model.md)) to avoid a parallel object graph for what is, in every case, a property of one specific observation. The standard this field is held to (enforced by review, not by the type system, since "good text" isn't machine-checkable):

- **Actionable** — describes what to do, not just what was wrong. "Consider widening the BPM range in the NM category" is a recommendation; "BPM variance is low" is not — that's a `Description`/`Reason`, not a `Recommendation`.
- **Specific to the finding's scope** — a stage-scoped finding's recommendation references that stage, not the tournament in the abstract.
- **Not prescriptive beyond the analyzer's competence.** An analyzer detecting low mapper diversity can recommend diversifying mappers; it should not recommend a specific replacement beatmap — that requires beatmap-search capability outside the Analysis Engine's responsibility.

## What Phase 5 deliberately does not include

- **No concrete domain analyzers** — Phase 5's job was the engine and contract, not the analyzers themselves; `engine_test.go`'s demo analyzers (`stageCoverageAnalyzer`, `beatmapPingAnalyzer`, `brokenAnalyzer`) exist only to prove the contract holds, and are test-only code, not shipped plugins. Phases 6–8 have since shipped the real plugins against this contract with zero changes to `engine.go`, `scope.go`, or `sourcehash.go` — Metadata Analyzers ([docs/10-metadata-analyzers.md](10-metadata-analyzers.md)), Pattern Analyzers ([docs/11-pattern-analyzers.md](11-pattern-analyzers.md)), and Tournament Analyzers ([docs/12-tournament-analyzers.md](12-tournament-analyzers.md)). This document's scope remains the engine/interface spec only — see those three docs for what the concrete analyzers actually check and why.
- **No Postgres-backed `Analysis`/`Finding` repository.** Same deferral as Phase 4's `BeatmapRepository` — the in-memory pattern is sufficient until a phase needs real persistence.
- **No cross-analyzer orchestration or dependency resolution.** Analyzers are flat and independent by design; there is no mechanism (and none is planned) for one analyzer to declare a dependency on another's output. If two analyzers need the same derived value, each computes it independently from `Input` — duplication here is the cost of the independence guarantee, and is preferred over coupling.

## Testing

`backend/internal/analysis/engine_test.go`, 6 tests:

- Independent analyzers run across their correct scopes, including beatmap dedup-by-ID across stages.
- A coverage analyzer's findings carry all required fields and a correctly computed score.
- One analyzer's invalid result doesn't block another analyzer's valid results from being returned.
- Duplicate analyzer names are rejected at registration.
- `SourceHash` is stable across repeated runs of unchanged input, changes when the relevant scope's data changes, and does **not** change for an unrelated sibling scope.

Run with:

```sh
cd backend && go test ./...
```
