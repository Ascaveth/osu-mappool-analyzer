# Metadata Analyzer Specifications

Phase 6 deliverable: the first concrete plugins built on the [Analysis Engine](09-analysis-engine-specification.md), operating on the metadata fields already produced by the [import pipeline](08-beatmap-import-pipeline.md). Implemented in `backend/internal/analysis/metadata`.

## Analyzers

| Analyzer | Scope | Single responsibility |
|---|---|---|
| `DifficultySettingsAnalyzer` | Beatmap | Validate AR/OD/CS/HP fall within osu!'s [0, 10] range |
| `ObjectDensityAnalyzer` | Beatmap | Expose object count / length / objects-per-second; flag a specific data-quality failure |
| `BPMRangeAnalyzer` | Category | Expose BPM spread across a category's filled slots; flag zero-diversity |
| `MapperRepetitionAnalyzer` | Category | Expose mapper distribution; flag single-mapper dominance |

### DifficultySettingsAnalyzer (Beatmap scope)

Reads `AR`, `OD`, `CS`, `HP` and reports each as a metric. Raises a **Critical** finding per field that falls outside `[0, 10]` — osu!'s own documented valid range, not a tournament-design opinion. This is a data-quality check: an out-of-range value almost always means the `.osu` file is malformed or the parser misread it, not that the mapper chose an unusual setting. `Score` is `1.0` if all four values are valid, `0.0` if any are not.

This analyzer deliberately does **not** judge whether a given AR/OD/CS/HP combination is "good" for tournament play — that requires comparison against sibling beatmaps' settings and tournament-design conventions this analyzer has no basis to assume. A future Balance Analyzer (Phase 8) is the right place for that judgment.

### ObjectDensityAnalyzer (Beatmap scope)

Exposes `object_count`, `length_seconds`, `slider_ratio`, and (when computable) `objects_per_second`. Raises a **Warning** when a beatmap has hit objects but a computed length of zero — that combination can only happen if the import pipeline's length derivation failed (see [08-beatmap-import-pipeline.md](08-beatmap-import-pipeline.md)), so it's always worth surfacing regardless of what "appropriate density" means for any given map.

### BPMRangeAnalyzer (Category scope)

Reports `bpm_min`, `bpm_max`, `bpm_range`, `bpm_mean`, and `filled_slots` for a category. Raises a **Warning** only when every filled beatmap in the category shares the *exact same* BPM (`bpm_range == 0`, with more than one slot filled) — an objectively zero-diversity state, not a threshold someone picked. Any non-zero range is reported as a metric without a finding, since judging what counts as "enough" BPM diversity is exactly the kind of subjective threshold this analyzer avoids inventing (see [Design Philosophy](#design-philosophy) below).

### MapperRepetitionAnalyzer (Category scope)

Reports `distinct_mappers` and `top_mapper_share` for a category. Raises a **Warning** when one mapper supplies more than 50% of a category's filled slots (with more than one slot filled). The 50% line is principled, not arbitrary: below it, no single mapper outweighs everyone else combined; at or above it, one mapper's style necessarily does. This directly implements the "overused mappers" validation case named in CLAUDE.md.

## Design philosophy: report, don't invent thresholds

Three of the four analyzers above could have been built around made-up magic numbers — "BPM range should be at least 20", "no mapper should exceed 30% of a category", "maps shorter than 60s are too short." None of those thresholds exist anywhere in the project's source documents, and asserting them as fact would be presenting an unfounded opinion as analysis output, which violates the spirit of [Architecture Principle 9](04-architecture-principles.md#9-reports-speak-in-conclusions-not-raw-numbers) (every conclusion must be explainable and defensible, not just confident-sounding).

The standard applied instead: **a Warning/Critical finding only fires when the underlying condition is objectively true regardless of tournament context** (all values identical → zero diversity; one mapper supplies a majority → mathematically dominant by definition; a setting outside `[0,10]` → not a valid osu! beatmap). Everything else is exposed as a `Metric` for a human, a report (Phase 9), or a better-justified future analyzer to interpret with real context this analyzer doesn't have.

## What is not implemented in this phase

- **Star Rating analyzer — not implemented.** `domain.Beatmap.StarRating` is left at `0` by the import pipeline (see [08-beatmap-import-pipeline.md](08-beatmap-import-pipeline.md#what-is-explicitly-deferred)); osu!'s star rating algorithm is a substantial strain-based computation over the full hit-object sequence that hasn't been built yet. A Star Rating analyzer is blocked on that calculator existing, not on analyzer design — once it's available, a `DifficultyRatingAnalyzer` slots into this same package with zero changes to the Engine, by construction of the plugin architecture.
- **Genre analyzer — not implemented, and not planned for this pipeline.** Genre is not present in the `.osu` file format at all; it's metadata osu!'s own website/API attaches separately. The import pipeline (Phase 4) only ever reads local `.osu` files, so there is no source data for a Genre analyzer to read. Adding one would require a second data source (an osu! API integration), which is a Data Collection concern (a new pipeline), not a Metadata Analysis concern — out of scope here.
- **Drain Time vs. Total Length distinction.** The import pipeline does not parse `[Events]` break periods, so `LengthSeconds` is total playable span (first object to last object end), not drain time excluding breaks. `ObjectDensityAnalyzer`'s `length_seconds` metric should be read with that caveat until break-period parsing is added to the import pipeline.
- **Artist analyzer — folded into existing metrics, not a standalone plugin.** `Artist` is already captured on `domain.Beatmap` and available to any analyzer or report; a dedicated "artist diversity" analyzer is deferred to Phase 8 (Diversity Analyzer), where it belongs alongside song/pattern diversity as one cohesive cross-cutting concern rather than a fourth near-duplicate of `BPMRangeAnalyzer`'s "count distinct values" shape.

## Testing

`backend/internal/analysis/metadata/metadata_test.go`, 10 tests:

- Per-analyzer happy path and finding-triggering cases (out-of-range difficulty settings, zero-length-with-objects, identical BPM, dominant mapper).
- Negative cases proving no false positives on diverse/valid data.
- An empty-category edge case (no panics, zero-value metrics).
- An integration test registering all four analyzers into a real `analysis.Engine` and running them together, confirming the expected number of independent Analyses are produced with no cross-analyzer interference.

Run with:

```sh
cd backend && go test ./...
```
