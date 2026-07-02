# Tournament Analyzer Specifications

Phase 8 deliverable: the cross-cutting analyzers that synthesize Phase 6/7's per-beatmap metrics into stage- and tournament-level judgments about composition, progression, and balance. Implemented in `backend/internal/analysis/tournament`.

## Analyzers

| Analyzer | Scope | Question it answers |
|---|---|---|
| `CompositionAnalyzer` | Stage | Is this stage's slot/mapper distribution lopsided across categories? |
| `ProgressionAnalyzer` | Tournament | Does average OD increase across stages without regressing or spiking? |
| `BalanceAnalyzer` | Category | Does each category have variation across AR, OD, and slider ratio? |
| `DiversityAnalyzer` | Stage | Are BPM, mappers, and songs sufficiently varied across a whole stage? |
| `SkillCoverageAnalyzer` | Stage | Is the stage's mechanical skillset coverage (aim, stream, tech, jump, alt) balanced and complete? |

Each one operates at a different scope from its Phase 6/7 counterpart, by design: Phase 6/7 analyzers look *within* one beatmap or one category; these look *across* categories or stages, catching problems no single-category analyzer is positioned to see.

### CompositionAnalyzer (Stage scope)

The stage-level counterpart to Phase 6's `MapperRepetitionAnalyzer` (which only ever looks within one category). Reports `category_count`, `total_slots`, `filled_slots`, `max_category_share`, and `distinct_mappers` for an entire stage. Two findings, both following the same "objective majority" pattern established in Phase 6:

- **Category imbalance** — Warning when one category holds more than 50% of the stage's total slots.
- **Single-mapper stage** — Warning when every filled slot across the *entire* stage (not just one category) was mapped by the same person.

### ProgressionAnalyzer (Tournament scope)

Computes average **Overall Difficulty (OD)** per stage, in `Stage.Order` sequence, and checks two things:

- **Regression** — Warning for any stage whose average OD is lower than the previous stage's. Each occurrence is its own finding, naming both stages.
- **Spike** — Warning for any single stage-to-stage OD increase that exceeds 2x the tournament's median increase (only evaluated with 4+ stages, i.e. 3+ deltas, since a median of fewer than 3 values isn't a meaningful baseline). The 2x multiplier is a named, explainable outlier heuristic (loosely related to interquartile-range fencing), not a measured fact — it's a constant (`spikeMultiplier`) specifically so it can be revisited.

**Why OD, not Star Rating:** the roadmap's progression question is fundamentally about difficulty, and Star Rating is the correct signal for that — but it isn't computed by the import pipeline yet (deferred in [Phase 4](08-beatmap-import-pipeline.md#what-is-explicitly-deferred), still not built as of Phase 8). Rather than block this analyzer on a difficulty calculator that doesn't exist, or silently use a metadata field as if it were difficulty, `ProgressionAnalyzer` uses OD (Overall Difficulty) explicitly and names it as such everywhere — in its findings, its metrics (`avg_od_stage_order_N`), and its docs. This is an honest scoping decision, not a difficulty calculator in disguise: when Star Rating becomes available, a `DifficultyProgressionAnalyzer` can be added alongside this one without modifying it, and reports can present both.

`Score` is `1.0 - (regressions / total transitions)` — `nil` when there's fewer than 2 stages with data to compare.

### BalanceAnalyzer (Category scope)

The tournament-quality counterpart to Phase 6's `BPMRangeAnalyzer`, which only ever checked BPM. `BalanceAnalyzer` checks three different axes — **AR**, **OD**, and **slider ratio** — each independently, using the same zero-variance convention as `BPMRangeAnalyzer`: a Warning fires only when every beatmap in the category shares the *exact same* value on that axis (with more than one slot filled). Slider ratio (proportion of objects that are sliders, already computed by Phase 4's normalizer) is used as the closest available proxy for "tap-heavy vs. slider-heavy" mapping emphasis — the closest thing to a "is one skill overrepresented" signal available without a pattern-classification model.

### DiversityAnalyzer (Stage scope)

Reports BPM, mapper, and song diversity across an **entire stage** (all categories combined), catching duplication that's invisible to a single-category view — a stage can have perfectly fine within-category diversity while still reusing the same song or mapper across two different categories. The one finding it raises is fully objective: a **duplicate song** (identical Artist+Title, exact string match) appearing in more than one slot within the same stage — directly implementing CLAUDE.md's "Duplicate characteristics" validation example.

### SkillCoverageAnalyzer (Stage scope)

Classifies each filled slot's beatmap into one or more skillset tags (`aim`, `jump`, `stream`, `tech`, `alt`, or `unclassified`) using `pattern.ComputeSkillsetProfile` — a shared pure function in `internal/analysis/pattern` that recomputes the same jump-distance, stream/burst-run, and slider-complexity primitives `JumpDistanceAnalyzer`/`StreamBurstAnalyzer`/`SliderComplexityAnalyzer` already report independently, rather than calling those analyzers directly (analyzers never depend on each other; see [docs/09-analysis-engine-specification.md](09-analysis-engine-specification.md)). Reports `filled_slots`, `distinct_skillsets`, `skillset_count_<tag>` per present tag, and `max_skillset_share`. Two findings:

- **Skillset overload** — Warning when one skillset's share of filled slots exceeds 60% (`skillsetMajorityThreshold`, set above `categoryMajorityThreshold` since skillset tags don't partition filled slots the way categories do — a beatmap can match more than one rule).
- **Missing skillset** — Warning when a taxonomy skillset has zero representation in a stage with at least 3 filled slots (`minSlotsForCoverageJudgment`, avoiding false positives on stages too small to judge against a multi-item taxonomy).

This directly targets the most common real-world "unbalanced"/"not fun" mappool complaint found in community feedback (osu! tournament forums, `poolingcore`, established mappooling guides): pools skewing toward one skillset systematically disadvantage players weak in that one area every match, regardless of how varied the pool's raw difficulty numbers are — a distinct problem from `BalanceAnalyzer`'s numeric AR/OD/slider-ratio variance check.

**Taxonomy is a named default, not a hardcoded assumption:** `DefaultTaxonomy()` ships as documented, revisitable Go constants (mirroring `spikeMultiplier`, `streamMinLength`, `categoryMajorityThreshold`), but `SkillCoverageAnalyzer.Taxonomy` is a public field a caller can override entirely. Full per-tournament user-editable taxonomy — a `domain.Tournament`-level field configurable through the API/UI — is an explicit deferred follow-up: no per-tournament settings surface exists in the domain model today, and this codebase's established precedent is to ship a named default first rather than build configurability speculatively.

## Why there is no ValidationAnalyzer

The roadmap names a fifth Tournament Analyzer, Validation, detecting "difficulty spikes, repeated map styles, weak progression, duplicate characteristics, missing skill coverage." A standalone `ValidationAnalyzer` is **not implemented**, and this isn't an oversight — it would directly contradict a decision already made in Phase 2:

> "Validation is not a separate entity — it's a `Finding` whose severity is `warning`/`critical` rather than `info`." ([docs/06-domain-model.md](06-domain-model.md#design-decisions-made-in-this-phase))

Every "validation" item the roadmap names is already produced by one of the five analyzers above, as an ordinary Warning-severity `Finding`:

| Roadmap validation item | Where it's actually produced |
|---|---|
| Difficulty spikes | `ProgressionAnalyzer`'s spike finding |
| Weak progression | `ProgressionAnalyzer`'s regression finding |
| Duplicate characteristics | `DiversityAnalyzer`'s duplicate-song finding |
| Repeated map styles | `BalanceAnalyzer`'s zero-variance findings (AR/OD/slider ratio) |
| Missing skill coverage | `SkillCoverageAnalyzer`'s missing-skillset and skillset-overload findings |

Building a sixth analyzer to re-detect the same conditions under a "Validation" label would create exactly the two-trees problem the Phase 2 decision was written to avoid: every Warning is already a validation result by construction, so a parallel pass would either duplicate logic or just relabel existing findings, neither of which adds analytical value.

## Testing

`backend/internal/analysis/tournament/tournament_test.go`, 18 tests:

- `CompositionAnalyzer`: dominant-category detection, single-mapper-stage detection, with a case confirming a balanced split (50/50) does *not* false-positive.
- `ProgressionAnalyzer`: regression detection, spike detection, a fully-monotonic sequence producing no findings (Score 1.0), and an insufficient-data (single stage) case that doesn't panic.
- `BalanceAnalyzer`: all-three-axes zero-variance detection, and a varied-values case producing no findings.
- `DiversityAnalyzer`: cross-category duplicate song detection, and a distinct-songs case producing no findings.
- `SkillCoverageAnalyzer`: skillset-overload detection, missing-skillset detection, a too-few-slots case suppressing the missing-skillset finding, mixed-skillset filled-slot counting, a custom-`Taxonomy` override test, and a no-filled-slots case that doesn't panic.
- An integration test running all five analyzers through a real `analysis.Engine` together across a 3-stage tournament.

`backend/internal/analysis/pattern/skillset_test.go` covers `ComputeSkillsetProfile` directly: wide-jump distance, stream-run detection, slider anchor/reverse-ratio computation, and zero-BPM/empty-HitObjects cases that must not panic.

Run with:

```sh
cd backend && go test ./...
```
