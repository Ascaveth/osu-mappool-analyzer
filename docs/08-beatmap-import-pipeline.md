# Beatmap Import Pipeline Specification

Phase 4 deliverable: how a `.osu` file becomes a `domain.Beatmap` (see [06-domain-model.md](06-domain-model.md)). Implemented in `backend/` (Go, per [05-stack-proposal.md](05-stack-proposal.md)).

## Pipeline

```
.osu file bytes
       │
       ▼
osufile.Parse           — format-faithful parsing, no domain knowledge
       │  (osufile.RawBeatmap)
       ▼
normalize.Beatmap        — derives domain metrics, computes hash
       │  (domain.Beatmap)
       ▼
storage.BeatmapRepository.Save  — dedupes by content hash, persists
```

This mirrors the Raw Data → Normalization stage of the architecture's core pipeline ([04-architecture-principles.md](04-architecture-principles.md#2-the-core-pipeline-is-fixed-everything-inside-it-is-pluggable)). `osufile` and `normalize` are separate packages on purpose: `osufile` knows the file format and nothing about the domain; `normalize` knows the domain and nothing about file syntax. A future second source format (e.g. a `.osz` archive, or an API-sourced beatmap) only needs its own raw parser feeding the same `normalize.Beatmap` entry point.

## Package layout

```
backend/
├── go.mod                                   github.com/Ascaveth/osu-mappool-analyzer/backend
├── internal/
│   ├── domain/
│   │   └── beatmap.go                       Beatmap, TimingPoint, HitObject
│   ├── osufile/
│   │   ├── types.go                         RawBeatmap and raw line types
│   │   ├── parser.go                        Parse(io.Reader) (*RawBeatmap, error)
│   │   └── parser_test.go
│   ├── normalize/
│   │   ├── beatmap.go                       Beatmap(*osufile.RawBeatmap, []byte) (*domain.Beatmap, error)
│   │   └── beatmap_test.go
│   └── storage/
│       ├── beatmap_repository.go            BeatmapRepository interface
│       └── memory/
│           ├── beatmap_repository.go        in-memory implementation (tests, local dev)
│           └── beatmap_repository_test.go
```

A Postgres-backed `storage.BeatmapRepository` implementation is deferred to whichever phase first needs real persistence (Phase 5/10) — the interface is the contract; `memory` is sufficient until then, per [Architecture Principle 11](04-architecture-principles.md#11-testability-is-a-design-constraint-not-an-afterthought) (analyzers and pipeline stages must be testable without standing up infrastructure).

## What gets parsed

`osufile.Parse` reads:

- **Header** — `osu file format vNN`; anything else returns `ErrNotAnOsuFile`. A UTF-8 BOM before the header is tolerated.
- **`[General]`, `[Metadata]`, `[Difficulty]`** — `Key: Value` lines, captured as string maps. Normalization picks the specific fields it needs (`Title`, `Artist`, `Creator`, `Version`, `Tags`, `HPDrainRate`, `CircleSize`, `OverallDifficulty`, `ApproachRate`, `SliderMultiplier`); unrecognized keys are preserved in the raw map but currently unused.
- **`[TimingPoints]`** — each line's offset, beat length, meter, and inherited/uninherited flag.
- **`[HitObjects]`** — each line's position, time, and type-specific fields (slider curve length + repeat count; spinner end time).

Malformed individual lines are skipped, not fatal — a single corrupt timing point or hit object line must not fail the entire import, per the parser's documented behavior (verified by `TestParse_SkipsMalformedLinesWithoutFailing`).

## What normalization computes

- **AR fallback** — format versions before AR existed (pre-v8) have no `ApproachRate` field; normalization falls back to `OverallDifficulty`, matching osu!'s own documented behavior.
- **Slider end time** — implements osu!'s actual slider duration formula: `duration = pixelLength × slides / (100 × sliderMultiplier × velocity) × beatLength`, where `velocity` comes from the nearest preceding inherited (green) timing point and resets to `1.0` at every uninherited (red) line. This is the single most error-prone part of `.osu` parsing and is covered by `TestBeatmap_HappyPath`'s slider duration assertion.
- **Dominant BPM** — a beatmap can have many timing changes; `domain.Beatmap.BPM` is the BPM that covers the most playtime (duration-weighted mode across uninherited points), not just the first or most frequent value. This single number is what BPM-distribution analyzers (Phase 6) will consume directly; the full `TimingPoints` list remains available for analyzers that need finer detail (e.g. rhythm/timing-variation analyzers in Phase 7).
- **Length** — earliest hit-object start to latest hit-object end (correctly accounting for slider/spinner duration, not just the last object's start time).
- **Content hash** — `sha256` of the exact source bytes (not a re-serialization), used by `BeatmapRepository` to detect re-imports of an identical file and resolve to the existing record rather than duplicating it, per the domain model's dedup rule.

## What is explicitly deferred

- **Star Rating is not computed.** `domain.Beatmap.StarRating` exists in the schema but is left at zero by this pipeline. osu!'s star rating algorithm is a substantial standalone computation (strain-based difficulty calculation over the full hit-object sequence) — it belongs with Phase 6 (Metadata Analyzers) as its own unit of work, not bundled into basic import. Treating import and difficulty-calculation as separate concerns also keeps the import pipeline fast and side-effect-free.
- **Mania hold notes** (and any other non-circle/slider/spinner object type) are parsed as `RawHitObjectUnknown` and dropped during normalization, not counted, not erroring. The project is std-pool-focused per [02-scope.md](02-scope.md); mode-specific parsing is a future addition if mania/taiko/catch pools ever enter scope.
- **No persistence implementation beyond in-memory.** The `BeatmapRepository` interface is final for this phase; a Postgres implementation is straightforward to add later without touching `osufile` or `normalize`, since both are storage-agnostic.

## Testing

12 tests across three packages, all passing:

- `osufile`: happy-path parse against a representative sample file (3 timing points incl. one inherited, circle + slider + spinner), explicit non-`.osu`-file rejection, empty-input rejection, malformed-line tolerance, BOM tolerance.
- `normalize`: happy-path metric derivation (AR/OD/CS/HP, BPM dominance across a BPM change, slider duration formula, length, slider ratio), missing required `[Difficulty]` field rejection, hash determinism, empty-hit-object-list safety (no panics on a pool with no objects yet).
- `storage/memory`: save + lookup by ID and by hash, dedup-on-save behavior, not-found errors.

Run with:

```sh
cd backend && go test ./...
```
