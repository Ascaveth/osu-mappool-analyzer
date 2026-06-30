# Pattern Analyzer Specifications

Phase 7 deliverable: analyzers operating on a beatmap's `HitObject` sequence — geometry, rhythm, and object-shape detail — rather than the pass-through metadata fields Phase 6 covered. Implemented in `backend/internal/analysis/pattern`.

## Parser extension

Phase 4's import pipeline didn't extract slider curve data because Phase 6's metadata analyzers didn't need it. Phase 7 does, so `osufile.RawHitObject` and `domain.HitObject` gained one field: **`CurvePointCount`** — the number of anchor points in a slider's path, parsed from the `curveType|x1:y1|x2:y2|...` field that was previously read only for its slider length/repeat count. The full curve geometry (anchor coordinates, curve type B/L/P/C) still isn't modeled — `CurvePointCount` is a complexity *proxy*, not a shape classifier. This is a deliberate, narrow extension: just enough to support `SliderComplexityAnalyzer` below, not a general curve-geometry engine.

A second pre-existing limitation matters more for this phase than it did for Phase 6: **`domain.HitObject.X/Y` is always an object's start position.** Slider end positions aren't tracked. Jump distance/angle analyzers below therefore measure start-to-start movement between objects, which is the conventional simplification used when full cursor-path tracking isn't implemented — but it is an approximation, not true cursor-to-cursor distance, and is documented as such at each call site.

## Analyzers

| Analyzer | Scope | Category | What it does |
|---|---|---|---|
| `JumpDistanceAnalyzer` | Beatmap | Geometry | Distance between consecutive landing objects |
| `JumpAngleAnalyzer` | Beatmap | Geometry | Turning angle at each interior landing object |
| `StreamBurstAnalyzer` | Beatmap | Rhythm | Detects and classifies runs of closely-spaced circles |
| `SliderComplexityAnalyzer` | Beatmap | Objects | Slider anchor count, reverse-slider usage |
| `SpinnerUsageAnalyzer` | Beatmap | Objects | Spinner count, duration, density |

All five are metrics-first, like Phase 6's analyzers: a finding only fires for an objectively-malformed-data case, never for a pattern characteristic that's merely unusual.

### JumpDistanceAnalyzer / JumpAngleAnalyzer (geometry)

Walk a beatmap's **landing objects** — circles and slider starts, in time order, with **spinners excluded**. A spinner has no fixed exit position (it ends wherever spin direction and timing happen to leave the cursor), so including it in a distance/angle calculation would measure noise, not map design. `JumpDistanceAnalyzer` reports min/max/avg distance in osu!pixels between consecutive landing objects. `JumpAngleAnalyzer` reports the turning angle (0-180°) at each interior landing object, computed via the law of cosines between the incoming and outgoing movement vectors, plus a `sharp_turn_count` of angles under 90° (an acute turn — the cursor reverses more than it continues forward, a geometric fact, not a difficulty judgment). Angles are skipped (not zero) when two consecutive objects share a position (a stacked note, common within streams), since the turning angle is undefined there, not zero.

Neither analyzer raises findings — there is no objectively "wrong" jump distance or angle. Whether a given distribution suits a category's intended difficulty is a Balance Analyzer judgment (Phase 8), made with sibling-beatmap context this analyzer doesn't have.

### StreamBurstAnalyzer (rhythm)

Classifies runs of circles whose inter-onset interval is at or below **1/4 beat (16th-note) snap**, with 15% timing tolerance, using the beatmap's dominant BPM (`domain.Beatmap.BPM`, computed by Phase 4's normalizer). A run of 3-6 such circles is a **burst**; 7 or more is a **stream**. These specific length cutoffs are a **stated convention** commonly used in the osu! mapping community for explainability, not an objective property of the beatmap — unlike Phase 6's zero-variance/majority-share conditions, "is 7 notes a stream or still a burst" has no universally correct answer. The convention is named in code (`streamMinLength`, `burstMinLength` constants) precisely so it can be revisited without anyone mistaking it for a measured fact. The analyzer never raises a finding: detecting a stream isn't itself a problem, it's a normal and often deliberate mapping choice.

### SliderComplexityAnalyzer / SpinnerUsageAnalyzer (objects)

`SliderComplexityAnalyzer` reports `slider_count`, `avg_anchor_count`, and `reverse_slider_ratio` (proportion of sliders with at least one repeat). `SpinnerUsageAnalyzer` reports `spinner_count`, `total_spinner_duration_seconds`, and `spinner_density` (spinner time as a fraction of the beatmap's total length). Both follow the same data-quality-only finding philosophy established in Phase 6: a slider with **zero curve anchor points** (a slider with no path at all — only possible from malformed/misparsed source data) and a spinner with **non-positive duration** (end time at or before start time — likewise only possible from malformed data) each raise a Warning. Neither analyzer judges whether a beatmap's *amount* of slider complexity or spinner usage is appropriate.

## What is not implemented in this phase

- **Cross-screen movement and "flow"/"precision" as holistic scores.** The roadmap names these under Movement, but they're aggregate judgments over a full pattern sequence (e.g. "does this map favor smooth, continuous motion or sharp direction reversals") rather than single measurable quantities. `JumpAngleAnalyzer`'s `sharp_turn_count` and `avg_angle_degrees` are the measurable primitives a future flow-scoring analyzer would build on — that synthesis is deferred until there's a defensible basis for combining them (see [Architecture Principle 9](04-architecture-principles.md#9-reports-speak-in-conclusions-not-raw-numbers): a flow "score" must mean something, not just exist).
- **True cursor-path distance for sliders.** Because slider end positions aren't modeled (see Parser extension above), `JumpDistanceAnalyzer` measures from a slider's *start* to the next object, not from where the cursor actually leaves the slider. Closing this gap requires parsing full curve geometry (anchor coordinates + curve type) and computing the actual endpoint, which is a larger addition than this phase's scope justified — `CurvePointCount` alone was sufficient for `SliderComplexityAnalyzer`.
- **Rhythm complexity / timing variation as a standalone analyzer.** `StreamBurstAnalyzer` covers the single most concrete, well-defined rhythm pattern (snap-based runs). General "rhythm complexity" (e.g. syncopation, polyrhythm detection) doesn't have an equally crisp, conventional definition to build against without inventing one — deferred rather than guessed at.

## Testing

`backend/internal/analysis/pattern/pattern_test.go`, 15 tests:

- `JumpDistanceAnalyzer`: distance computation, spinner exclusion, fewer-than-two-object edge case.
- `JumpAngleAnalyzer`: a known right-angle case (asserted to ~90°), stacked-note skip behavior.
- `StreamBurstAnalyzer`: stream detection (8 notes), burst detection (4 notes), widely-spaced notes producing no runs, zero-BPM safety (no panic, no division by zero).
- `SliderComplexityAnalyzer` / `SpinnerUsageAnalyzer`: metric computation plus their respective malformed-data finding conditions, and empty-object-list edge cases.
- An integration test running all five analyzers through a real `analysis.Engine` together.

Run with:

```sh
cd backend && go test ./...
```
