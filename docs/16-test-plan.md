# Test Plan

Phase 12 deliverable: how the project verifies analyzer correctness (`pool-lab-plan.md`'s Phase 12 objective), what's covered today, and what's deliberately deferred. This is a living document — update the coverage table and gap list as packages change, don't let it drift into describing a codebase that no longer exists.

## Test pyramid for this project

```
        ▲
        │   1 integration test (internal/integration)
        │   parse -> normalize -> analyze -> report, one real run
        │
        │   57+ unit tests, one package per pipeline stage
        │   (osufile, normalize, domain, analysis/*, report, storage)
        ▼
```

This is unit-heavy by design, not by neglect. Every analyzer is a pure function over a `domain.Tournament`/`domain.Beatmap` (Architecture Principle 6: determinism; Principle 11: analyzer independence) — there's no I/O, no network, no shared mutable state inside an analyzer to require heavier test machinery. A small number of integration tests exist purely to prove the *wiring* between stages, not to re-verify logic the unit tests already cover. There is no end-to-end (E2E) layer (browser, running server) yet, because there is no running server or live-data frontend yet (see "Not in scope," below) — that layer gets added when one exists, not before.

## Coverage table

| Package | Test file(s) | Tests | Coverage | Covers |
|---|---|---|---|---|
| `internal/osufile` | `parser_test.go` | 8 | 88.6% | `.osu` parsing: happy path, malformed input, empty input, BOM tolerance, malformed-line skipping, slider curves + inherited timing points, extreme values (high BPM, dense objects), missing `[TimingPoints]` section |
| `internal/normalize` | `beatmap_test.go` | 7 | 91.0% | Raw → `domain.Beatmap`: metadata/difficulty extraction, slider duration math, dominant-BPM selection, missing required fields, extreme-value passthrough (no clamping), missing-timing-points fallback, empty-beatmap edge case |
| `internal/domain` | `configuration_test.go` | 7 | 100% | `ValidateConfiguration`: happy path, parallel same-order stages (not an error — see "Findings from this phase" below), duplicate category order (error), zero-slot category (error), duplicate category name within a stage (warning, not error), duplicate names across different stages (not flagged) |
| `internal/analysis` | `engine_test.go` | 5 | 70.6% | Plugin contract: independent analyzers run across all matching scopes, beatmap dedup by ID, required-Finding-fields enforcement, one analyzer's failure doesn't block others, duplicate-name registration rejected, deterministic `SourceHash` |
| `internal/analysis/metadata` | `metadata_test.go` | 10 | 93.3% | Mapper/BPM diversity and uniformity detection, dedup, boundary values |
| `internal/analysis/pattern` | `pattern_test.go` | 14 | 95.9% | Jump distance/angle, flow consistency, object density, density anomalies, cursor travel, tech factor |
| `internal/analysis/tournament` | `tournament_test.go` | 11 | 90.2% | Composition (category dominance), Progression (difficulty spikes), Balance, Diversity |
| `internal/report` | `report_test.go` | 4 | 94.2% | Summary narration, citation/counting, severity weighting, statistics aggregation |
| `internal/storage` | — (contract lives in `storagetest`) | — | n/a | Interface only; see below |
| `internal/storage/storagetest` | `beatmap_repository.go` | 6 subtests | n/a (helper) | Reusable `BeatmapRepository` contract: save/find by ID and hash, dedup by hash, not-found errors, record isolation |
| `internal/storage/memory` | `beatmap_repository_test.go` | 1 (runs the contract) | 100% | In-memory repository against the shared contract above |
| `internal/integration` | `pipeline_test.go` | 2 | n/a (cross-package) | Full pipeline composition (parse → normalize → analyze → report) against real `.osu` fixtures; invalid-configuration short-circuit before analysis runs |

Run locally: `cd backend && go test ./... -cover`. Run with verbose per-test output: add `-v`.

## What "happy path / edge case / invalid beatmap / invalid tournament configuration" means here

- **Happy path** — every package's `*_HappyPath` / baseline test: a well-formed `.osu` fixture, a complete tournament configuration, a clean pool with no findings.
- **Edge cases** — empty input (`TestParse_EmptyInput`), zero hit objects (`TestBeatmap_EmptyHitObjectsDoesNotPanic`), missing timing data (`TestParse_MissingTimingPointsSectionDoesNotFail`, `TestBeatmap_MissingTimingPointsFallsBackToDefaultBeatLength`), extreme values like 600+ BPM and dense object clusters (`TestParse_ExtremeValues`, `TestBeatmap_ExtremeBPMAndDenseObjectsNormalizeWithoutClamping`) — the normalize layer must report these as-is, never silently clamp or reject a value just because it's unusual.
- **Invalid beatmaps** — malformed `.osu` lines that must be skipped, not fail the whole import (`TestParse_SkipsMalformedLinesWithoutFailing`), and beatmaps missing a required `[Difficulty]` field, which *should* fail (`TestBeatmap_MissingRequiredDifficultyField`) — normalization tolerates noise but not missing data it can't safely default.
- **Invalid tournament configuration** — `internal/domain/configuration_test.go` and `internal/integration/pipeline_test.go`'s `TestPipeline_InvalidTournamentConfigurationIsCaughtBeforeAnalysis`. See "Findings from this phase" for what was actually built here.

## Findings from this phase

Writing these tests surfaced two real gaps, both fixed as part of Phase 12 rather than just documented around:

1. **`docs/07-tournament-configuration.md` defined validation rules that no code enforced.** The spec said cross-field rules like "`Stage.order` is unique" and "`slotCount` must be ≥ 1" are "enforced in application code," but no such code existed anywhere in `backend/`. `internal/domain/configuration.go` (`ValidateConfiguration`) now implements it.
2. **`docs/07` contradicted itself on stage ordering.** Its own "Validation rules" section said `Stage.order` must be unique, while its "Supporting future/non-linear formats" section — two sections later — named same-`order` stages as the explicit, intentional mechanism for parallel/concurrent stages (e.g. simultaneous group pools), "not as an error." `ValidateConfiguration` follows the more deliberate, explicitly-justified rule (same-order stages are allowed) and `docs/07`'s rule 1 wording was corrected to match, rather than silently picking one interpretation and leaving the doc wrong.

## Fixture strategy

`testdata/` directories hold small, hand-crafted `.osu` files — one fixture per scenario, never a real downloaded beatmap. This keeps fixtures readable (every line in `extreme_values.osu` exists to exercise one specific code path), avoids any licensing ambiguity around redistributing real map files, and keeps the test suite fast and deterministic. When a new parser/normalize edge case needs covering, add a new minimal fixture rather than growing an existing one to do double duty.

Current fixtures (`internal/osufile/testdata/`): `sample.osu` (happy path), `missing_difficulty.osu`, `slider_curves.osu` (bezier + perfect-circle sliders, inherited timing points), `extreme_values.osu` (high BPM, dense objects, long spinner), `missing_timing_points.osu`.

## Regression policy

No regression-specific fixtures exist yet because no bug has been found and fixed in this codebase to regress against. Going forward: when a bug is found, the fix PR must add a fixture/test that reproduces it *before* the fix, confirm it fails, then fix it — not just patch the code and move on. That test stays in the suite permanently and should say in a comment what it guards against.

## CI

`.github/workflows/backend-tests.yml` runs `go vet ./...` and `go test ./... -cover` on every push and PR that touches `backend/` or the workflow file itself. No matrix build — a single-module Go project at this stage doesn't need one; revisit if multi-version support ever becomes a requirement.

## What is explicitly not in scope for this phase

- **Live API contract tests.** Phase 10 validated `docs/api/openapi.yaml` with `@redocly/cli lint` and a Prism mock server, but there is no running server implementation (`docs/14-api-specification.md`'s scope note — no `TournamentRepository`/HTTP handlers exist). Contract tests against a real server are blocked on that server existing; add them when it does.
- **Frontend automated tests.** `frontend/` (Phase 11) renders `lib/sample-data.ts`, not live data — there's no real behavior to test yet beyond "does static sample data render," which a snapshot test would only encode as a brittle copy of the fixture. Add a real test suite (component tests against actual API responses, at minimum) once the frontend fetches from a running backend.
- **Load/performance testing.** Not called for by Phase 12's task list, and there's no deployed system yet to load-test.

## Testing checklist (for future analyzers)

When adding a new `analysis.Analyzer` implementation, per Architecture Principle 11 (analyzer independence) and the existing test files' pattern:

- [ ] Happy-path test against a clean, well-formed scope
- [ ] At least one test proving the analyzer produces zero findings when nothing is wrong (not every test should assert a finding)
- [ ] At least one test proving the analyzer's findings carry required `Severity`/`Reason`/`Recommendation` (the Engine enforces this, but the analyzer's own test should assert it directly too)
- [ ] Edge case(s) specific to that analyzer's domain (e.g. a single-slot category, a tournament with one stage)
- [ ] No test depends on another analyzer's output — construct `analysis.Input` directly from synthetic `domain.Tournament` data
