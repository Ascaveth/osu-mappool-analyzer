# Test Plan

Phase 12 deliverable: how the project verifies analyzer correctness (`pool-lab-plan.md`'s Phase 12 objective), what's covered today, and what's deliberately deferred. This is a living document — update the coverage table and gap list as packages change, don't let it drift into describing a codebase that no longer exists.

## Test pyramid for this project

```
        ▲
        │   2 integration tests (internal/integration)
        │   parse -> normalize -> analyze -> report, one real run;
        │   plus internal/api's HTTP-level integration test
        │
        │   150+ unit tests, one package per pipeline stage
        │   (osufile, normalize, domain, config, modmap, osuapi,
        │    enrich, analysis/*, report, api, storage)
        ▼
```

This is unit-heavy by design, not by neglect. Every analyzer is a pure function over a `domain.Tournament`/`domain.Beatmap` (Architecture Principle 6: determinism; Principle 11: analyzer independence) — there's no I/O, no network, no shared mutable state inside an analyzer to require heavier test machinery. A small number of integration tests exist purely to prove the *wiring* between stages, not to re-verify logic the unit tests already cover. `internal/api`'s tests are the one deliberate exception, since HTTP handler wiring genuinely needs `httptest` rather than a synthetic `domain.Tournament` call. There is still no browser-level E2E layer, because the frontend has no automated tests yet (see "Not in scope," below) — that layer gets added when one exists, not before.

## Coverage table

All packages pass (`go test ./...`, captured 2026-07-02). This table lists every package `go test ./... -cover` reports, including packages with no statements to cover and test-only helper packages.

| Package | Test file(s) | Tests | Coverage | Covers |
|---|---|---|---|---|
| `internal/osufile` | `parser_test.go` | 10 | 85.6% | `.osu` parsing: happy path, malformed input, empty input, BOM tolerance, malformed-line skipping, slider curves + inherited timing points, extreme values (high BPM, dense objects), missing `[TimingPoints]` section |
| `internal/normalize` | `beatmap_test.go` | 11 | 91.6% | Raw → `domain.Beatmap`: metadata/difficulty extraction, slider duration math, dominant-BPM selection, missing required fields, extreme-value passthrough (no clamping), missing-timing-points fallback, empty-beatmap edge case |
| `internal/domain` | `configuration_test.go` | 8 | 100% | `ValidateConfiguration`: happy path, parallel same-order stages (not an error — see "Findings from this phase" below), duplicate category order (error), zero-slot category (error), duplicate category name within a stage (warning, not error), duplicate names across different stages (not flagged) |
| `internal/config` | `config_test.go` | 1 (table-driven, 4 cases) | 66.7% | `Load()`: both env vars unset, both set, one-set/one-missing (error), default `PORT`/`ALLOWED_ORIGINS` — see [docs/18](18-configuration-and-modmap.md) |
| `internal/modmap` | `modmap_test.go` | 4 | 95.2% | `FromCategoryName` single/combo/unresolvable resolution, `IsFreeMod`, `FreeModCandidates` excludes DoubleTime, `AffectsStarRating` — see [docs/18](18-configuration-and-modmap.md) |
| `internal/osuapi` | `client_test.go` | 5 | 80.5% | osu! API HTTP client: OAuth token fetch/reuse, Star Rating fetch by beatmap+mods, checksum lookup, error propagation |
| `internal/osuapi/osuapitest` | — (test helper, no `_test.go`) | — | 0.0% (no test statements; it *is* the test double) | `FakeClient` used by `internal/enrich`'s tests |
| `internal/enrich` | `starrating_test.go` | 5 | 92.3% | Import-time Star Rating enrichment orchestration: ID resolution (parsed vs. checksum-lookup fallback), eager mod-combo fetch, partial-failure tolerance, total-failure error path |
| `internal/analysis` | `engine_test.go` | 5 | 71.4% | Plugin contract: independent analyzers run across all matching scopes, beatmap dedup by ID, required-Finding-fields enforcement, one analyzer's failure doesn't block others, duplicate-name registration rejected, deterministic `SourceHash` |
| `internal/analysis/metadata` | `metadata_test.go` | 10 | 93.3% | Mapper/BPM diversity and uniformity detection, dedup, boundary values |
| `internal/analysis/pattern` | `pattern_test.go`, `skillset_test.go` | 20 | 93.7% | Jump distance/angle, stream/burst detection, slider complexity, spinner usage, and `ComputeSkillsetProfile`'s shared skillset-classification primitives — see [docs/11](11-pattern-analyzers.md) for the exact metric set (there is no separate "flow consistency"/"cursor travel"/"tech factor" concept in this package; an earlier version of this table implied there was) |
| `internal/analysis/tournament` | `tournament_test.go`, `difficulty_spread_test.go` | 28 | 91.0% | Composition (category dominance), Progression (difficulty spikes), Balance, Diversity, Skill Coverage, Difficulty Spread (gap/spike detection, FreeMod ranging) |
| `internal/report` | `report_test.go` | 4 | 94.2% | Summary narration, citation/counting, severity weighting, statistics aggregation |
| `internal/api` | `api_integration_test.go`, `beatmaps_test.go`, `testhelpers_test.go`, `tournaments_test.go` | 22 | 64.1% | HTTP handlers: tournament/beatmap CRUD, import + enrichment wiring, CORS/logging middleware, end-to-end request/response contract |
| `internal/storage` | — (contract lives in `storagetest`) | — | n/a (interface only) | Interface only; see below |
| `internal/storage/storagetest` | `beatmap_repository.go` | 6 subtests | n/a (helper) | Reusable `BeatmapRepository` contract: save/find by ID and hash, dedup by hash, not-found errors, record isolation |
| `internal/storage/memory` | `beatmap_repository_test.go`, `star_rating_repository_test.go`, `tournament_repository_test.go` | 13 | 82.2% | In-memory `BeatmapRepository` against the shared contract above, plus `StarRatingRepository` and `TournamentRepository` implementations |
| `internal/integration` | `pipeline_test.go` | 2 | [no statements] | Full pipeline composition (parse → normalize → analyze → report) against real `.osu` fixtures; invalid-configuration short-circuit before analysis runs |
| `cmd/server` | — (composition root, no `_test.go`) | — | 0.0% | Wiring only (see [docs/18](18-configuration-and-modmap.md)); exercised indirectly via `internal/api` and `internal/integration` |

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

- **Frontend automated tests.** The frontend now fetches live data from the real API (`frontend/lib/api/rest.ts` — see [docs/15](15-ui-specification.md#backend-wiring)), so this gap is no longer "nothing real to test yet" — it's a genuine, un-backfilled hole. No `*.test.*`/`*.spec.*` files exist anywhere under `frontend/`. Add a real test suite (component tests against actual/mocked API responses, at minimum for the wizard's validation logic and the pool builder's import flow) as a follow-up.
- **Contract tests against a Postgres-backed implementation.** `internal/api`'s integration tests already exercise the real HTTP handlers end to end (see the coverage table above) against the current in-memory storage — that gap closed. What remains is verifying the same contract holds once `internal/storage/postgres` exists (see [docs/14](14-api-specification.md#storage)); blocked on that implementation, not on test design.
- **Load/performance testing.** Not called for by any shipped phase's task list, and there's no deployed system yet to load-test.

## Testing checklist (for future analyzers)

When adding a new `analysis.Analyzer` implementation, per Architecture Principle 11 (analyzer independence) and the existing test files' pattern:

- [ ] Happy-path test against a clean, well-formed scope
- [ ] At least one test proving the analyzer produces zero findings when nothing is wrong (not every test should assert a finding)
- [ ] At least one test proving the analyzer's findings carry required `Severity`/`Reason`/`Recommendation` (the Engine enforces this, but the analyzer's own test should assert it directly too)
- [ ] Edge case(s) specific to that analyzer's domain (e.g. a single-slot category, a tournament with one stage)
- [ ] No test depends on another analyzer's output — construct `analysis.Input` directly from synthetic `domain.Tournament` data
