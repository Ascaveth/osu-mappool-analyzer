# Metadata Analyzer Specifications

Phase 6 deliverable: the first concrete plugins built on the [Analysis Engine](09-analysis-engine-specification.md), operating on the metadata fields already produced by the [import pipeline](08-beatmap-import-pipeline.md). Implemented in `backend/internal/analysis/metadata`.

## Analyzers

| Analyzer | Scope | Single responsibility |
|---|---|---|
| `DifficultySettingsAnalyzer` | Beatmap | Validate AR/OD/CS/HP fall within osu!'s [0, 10] range |
| `ARCalibrationAnalyzer` | Beatmap | Flag AR that looks mismatched to the beatmap's own BPM (too tight or too loose a reading window) |
| `CSPrecisionAnalyzer` | Beatmap | Flag CS high enough to add a precision-difficulty spike Star Rating doesn't fully weight |
| `ObjectDensityAnalyzer` | Beatmap | Expose object count / length / objects-per-second; flag a specific data-quality failure |
| `BPMRangeAnalyzer` | Category | Expose BPM spread across a category's filled slots; flag zero-diversity |
| `MapperRepetitionAnalyzer` | Category | Expose mapper distribution; flag single-mapper dominance |

### DifficultySettingsAnalyzer (Beatmap scope)

Reads `AR`, `OD`, `CS`, `HP` and reports each as a metric. Raises a **Critical** finding per field that falls outside `[0, 10]` — osu!'s own documented valid range, not a tournament-design opinion. This is a data-quality check: an out-of-range value almost always means the `.osu` file is malformed or the parser misread it, not that the mapper chose an unusual setting. `Score` is `1.0` if all four values are valid, `0.0` if any are not.

This analyzer deliberately does **not** judge whether a given AR/OD/CS/HP combination is "good" for tournament play — that requires comparison against sibling beatmaps' settings and tournament-design conventions this analyzer has no basis to assume. A future Balance Analyzer (Phase 8) is the right place for that judgment.

### ARCalibrationAnalyzer (Beatmap scope)

Converts AR to osu!'s client-side approach-time formula (a fixed game constant, not a tournament convention) and compares it against the beatmap's own beat length (`60000 / BPM`) as a ratio of "beats of reading window per note". Raises a **Warning** when that ratio falls outside a named band (`arRatioLowThreshold`/`arRatioHighThreshold` in `ar_calibration.go`), skipping silently when `BPM <= 0`.

This directly implements the metadata-feedback framework's finding that AR complaints are almost always a BPM/pattern-density mismatch rather than an absolute-value fault: a low AR relative to a fast beatmap's note rate crowds multiple notes' approach circles on screen at once (a distinct reading-skill demand), and a high AR relative to a slow beatmap gives an unusually generous window — neither is inherently wrong (e.g. breakcore maps deliberately use AR to disambiguate 1/3 vs 1/4 rhythm), so findings here are a prompt to verify intent, not an assertion of fault.

### CSPrecisionAnalyzer (Beatmap scope)

Raises a **Warning** when CS crosses `csSpikeThreshold` (`cs_precision.go`). CS shrinks hitbox size, and its precision demand compounds with spacing/speed in a way Star Rating's aim component — driven mostly by cursor distance and velocity — doesn't fully weight. This gives staff a signal to playtest a beatmap's precision demand independently of where Star Rating places it in the pool.

### ObjectDensityAnalyzer (Beatmap scope)

Exposes `object_count`, `length_seconds`, `slider_ratio`, and (when computable) `objects_per_second`. Raises a **Warning** when a beatmap has hit objects but a computed length of zero — that combination can only happen if the import pipeline's length derivation failed (see [08-beatmap-import-pipeline.md](08-beatmap-import-pipeline.md)), so it's always worth surfacing regardless of what "appropriate density" means for any given map.

### BPMRangeAnalyzer (Category scope)

Reports `bpm_min`, `bpm_max`, `bpm_range`, `bpm_mean`, and `filled_slots` for a category. Raises a **Warning** only when every filled beatmap in the category shares the *exact same* BPM (`bpm_range == 0`, with more than one slot filled) — an objectively zero-diversity state, not a threshold someone picked. Any non-zero range is reported as a metric without a finding, since judging what counts as "enough" BPM diversity is exactly the kind of subjective threshold this analyzer avoids inventing (see [Design Philosophy](#design-philosophy) below).

**BPM is a pool-wide balance concern, not just a per-category one.** This exact-zero check is deliberately narrow; the softer, stage-wide "everything sits near the same tempo across categories" pattern the metadata-feedback framework calls a pool-balance concern is handled by `tournament.DiversityAnalyzer`'s BPM-clustering finding ([docs/12-tournament-analyzers.md](12-tournament-analyzers.md)) rather than duplicated here, since that analyzer already scans BPM across an entire Stage.

### MapperRepetitionAnalyzer (Category scope)

Reports `distinct_mappers` and `top_mapper_share` for a category. Raises a **Warning** when one mapper supplies more than 50% of a category's filled slots (with more than one slot filled). The 50% line is principled, not arbitrary: below it, no single mapper outweighs everyone else combined; at or above it, one mapper's style necessarily does.

## Design philosophy: report, don't invent thresholds

Three of the four analyzers above could have been built around made-up magic numbers — "BPM range should be at least 20", "no mapper should exceed 30% of a category", "maps shorter than 60s are too short." None of those thresholds exist anywhere in the project's source documents, and asserting them as fact would be presenting an unfounded opinion as analysis output, which violates the spirit of [Architecture Principle 9](04-architecture-principles.md#9-reports-speak-in-conclusions-not-raw-numbers) (every conclusion must be explainable and defensible, not just confident-sounding).

The standard applied instead: **a Warning/Critical finding only fires when the underlying condition is objectively true regardless of tournament context** (all values identical → zero diversity; one mapper supplies a majority → mathematically dominant by definition; a setting outside `[0,10]` → not a valid osu! beatmap). Everything else is exposed as a `Metric` for a human, a report (Phase 9), or a better-justified future analyzer to interpret with real context this analyzer doesn't have.

`ARCalibrationAnalyzer` and `CSPrecisionAnalyzer` are a deliberate, named exception to "objective-only": they are exactly the judgment-based follow-up this section originally deferred to "a future Balance Analyzer." Both use a named, documented threshold constant flagged in-code as an inference/calibration choice (the same convention `tournament.SkillCoverageAnalyzer`'s `jumpDistanceThreshold` and `tournament.DifficultySpreadAnalyzer`'s heuristics already establish), not an invented-and-hidden magic number — the distinction this section cares about is that a threshold be named, documented, and revisitable, not that every analyzer in this package stay purely objective forever.

## Why OD, Mapper pre-scout fairness, and Artist compliance have no dedicated check

The metadata-feedback framework this section responds to explicitly separates **gameplay-setting metadata** (AR, OD, CS, BPM) from **provenance metadata** (Mapper, Artist/Music), and further separates OD from AR/CS/BPM within the gameplay-setting group. Three items from that framework are deliberately *not* new analyzer code:

- **OD.** OD acts on score margin (the accuracy/scoring window), not as a playability gate the way AR/CS/BPM do — it rarely surfaces as a standalone playability complaint, HR already ceilings it, and ranked-mapping convention standardizes NM OD into a narrow band. Where OD genuinely causes a problem, it shows up as a *scoring-fairness* issue (e.g. DT's hit-window compression) rather than a difficulty-calibration one — an axis this codebase has no model for, since no ScoreV2/hit-window simulation exists anywhere in the pipeline. `DifficultySettingsAnalyzer` still range-validates OD (a pure data-quality check, same as AR/CS/HP), but manufacturing an OD *difficulty* finding here would misclassify it onto the wrong rubric axis. A future `ScoringFairnessAnalyzer` is the right home for this, if hit-window/ScoreV2 simulation data ever becomes available.
- **Mapper custom-map pre-scout fairness.** Beyond clique/single-mapper domination (already covered by `MapperRepetitionAnalyzer`), the framework raises a fairness concern specific to custom/unranked maps: captains can't pre-scout them the way they can a ranked map, which raises favoritism/inside-knowledge risk. `domain.Beatmap` has no "custom vs. ranked" flag today (only `OsuBeatmapID *int64`, which is nil for locally-authored maps *and* for ranked maps whose ID simply wasn't parsed — the two cases aren't distinguishable), so there's no reliable signal to build this check on without a new domain field.
- **Artist/Music content-usage compliance.** The framework treats copyright/content-usage permission as the load-bearing Artist/Music concern — badged tournaments must comply with osu!'s content-usage permissions, and non-compliance can mean a mid-tournament mapset takedown. This needs an artist-permission dataset (which artists/songs are cleared for tournament use) that this codebase doesn't ingest from anywhere. The softer "aesthetic fatigue / song repetition" concern the framework treats as separate and secondary *is* already covered, by `tournament.DiversityAnalyzer`'s duplicate-song finding.

Both the Mapper and Artist gaps are the same category of limitation already named for tester/playtest feedback data in [docs/12-tournament-analyzers.md](12-tournament-analyzers.md): real, but blocked on a data source this project doesn't have yet, not an oversight.

## What is not implemented in this phase

- **Star Rating — implemented, but not as a local calculator and not in this package.** `domain.Beatmap.StarRating` is still left at `0` by the import pipeline (a strain-based difficulty calculator was never built, and per-mod Star Rating can't live on the immutable `Beatmap` aggregate anyway — see [08-beatmap-import-pipeline.md](08-beatmap-import-pipeline.md#what-is-explicitly-deferred)). Instead, real official Star Rating is fetched from the live osu! API at import time and stored per-(beatmap, mods) in `storage.StarRatingRepository`; `tournament.DifficultySpreadAnalyzer` ([docs/12-tournament-analyzers.md](12-tournament-analyzers.md)) consumes it through an injected `StarRatingLookup` interface, keeping this package's and that package's analyzers network-free. Full integration details (OAuth, client, enrichment pipeline, deferred scope) are in [docs/17-osu-api-integration.md](17-osu-api-integration.md).
- **Genre analyzer — not implemented, and not planned for this pipeline.** Genre is not present in the `.osu` file format at all; it's metadata osu!'s own website/API attaches separately. The import pipeline (Phase 4) only ever reads local `.osu` files, so there is no source data for a Genre analyzer to read. Adding one would require a second data source (an osu! API integration), which is a Data Collection concern (a new pipeline), not a Metadata Analysis concern — out of scope here.
- **Drain Time vs. Total Length distinction.** The import pipeline does not parse `[Events]` break periods, so `LengthSeconds` is total playable span (first object to last object end), not drain time excluding breaks. `ObjectDensityAnalyzer`'s `length_seconds` metric should be read with that caveat until break-period parsing is added to the import pipeline.
- **Artist analyzer — folded into existing metrics, not a standalone plugin.** `Artist` is already captured on `domain.Beatmap` and available to any analyzer or report; a dedicated "artist diversity" analyzer is deferred to Phase 8 (Diversity Analyzer), where it belongs alongside song/pattern diversity as one cohesive cross-cutting concern rather than a fourth near-duplicate of `BPMRangeAnalyzer`'s "count distinct values" shape.

## Testing

`backend/internal/analysis/metadata/metadata_test.go`, 17 tests:

- Per-analyzer happy path and finding-triggering cases (out-of-range difficulty settings, AR/BPM mismatch in both directions, high CS, zero-length-with-objects, identical BPM, dominant mapper).
- Negative cases proving no false positives on diverse/valid data.
- An empty-category edge case (no panics, zero-value metrics) and a zero-BPM edge case for `ARCalibrationAnalyzer` (no panics, no finding).
- An integration test registering all six analyzers into a real `analysis.Engine` and running them together, confirming the expected number of independent Analyses are produced with no cross-analyzer interference.

`tournament.DiversityAnalyzer`'s BPM-clustering finding is tested alongside the rest of that analyzer in `backend/internal/analysis/tournament/tournament_test.go` (see [docs/12-tournament-analyzers.md](12-tournament-analyzers.md)).

Run with:

```sh
cd backend && go test ./...
```
