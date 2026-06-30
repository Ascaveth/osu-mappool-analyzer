# Glossary

Terms used throughout the codebase, docs, and reports. Where a term has both a generic osu! meaning and a project-specific one, the project-specific meaning is given.

## Tournament structure

- **Tournament** — A complete competitive event with a user-defined structure of stages.
- **Stage** — A round within a tournament (e.g. Qualifiers, Round of 16, Quarterfinals, Finals, Grand Finals). Stages are ordered and user-defined; no stage name or count is hardcoded.
- **Category** — A grouping of slots within a stage by mod or intent (e.g. NM, HD, HR, DT, FM, TB). Categories are user-defined per stage.
- **Slot** — A single beatmap position within a category (e.g. "NM3" is the 3rd No-Mod slot). A slot is filled by exactly one beatmap.
- **Pool** — The complete set of beatmaps assigned to a stage (all categories, all slots).
- **Mappool** — The complete set of all pools across all stages of a tournament. Used loosely; "pool" and "mappool" are often interchangeable in conversation but "pool" is the precise per-stage term in the domain model.

## Mod categories (common, not exhaustive — categories are user-defined)

- **NM** — No Mod
- **HD** — Hidden
- **HR** — Hard Rock
- **DT** — Double Time
- **FM** — Free Mod
- **TB** — Tiebreaker

## Beatmap data

- **Beatmap** — A single playable map: one song + one difficulty + its metadata, timing points, and hit objects. Treated as immutable source data.
- **Metadata** — Descriptive attributes of a beatmap: title, artist, mapper, BPM, AR, OD, CS, HP, star rating, length, etc.
- **Timing Point** — A definition of BPM/offset/signature at a point in a beatmap, used to derive rhythm-related metrics.
- **Hit Object** — A circle, slider, or spinner in a beatmap; the raw unit pattern analyzers operate on.
- **Star Rating (SR)** — osu!'s computed difficulty rating for a beatmap.

## Analysis Engine

- **Normalization** — The process of converting raw imported beatmap/tournament data into a consistent internal representation the Analysis Engine can consume.
- **Analyzer** — An independent module that accepts normalized data, performs one analytical responsibility, and produces findings, metrics, and recommendations. Analyzers never depend on each other.
- **Analysis** — The structured output of running one or more analyzers against a pool or tournament.
- **Finding** — A specific observation an analyzer produces (e.g. "NM category BPM range is unusually narrow").
- **Metric** — A quantified measurement produced by an analyzer (e.g. average star rating per stage).
- **Recommendation** — An actionable suggestion attached to a finding (e.g. "consider widening BPM range in NM category").
- **Severity** — A classification of how significant a finding is (e.g. info, warning, critical).
- **Validation** — The subset of analysis specifically concerned with detecting problems (as opposed to neutral metrics/observations).
- **Report** — A human-language document assembled from one or more analyses, explaining what happened, why it matters, and what to do about it.

## Cross-cutting concepts

- **Composition** — The makeup of a pool: how categories, mod types, and maps are distributed.
- **Progression** — How difficulty and other characteristics change across stages (e.g. Qualifiers → Grand Finals should generally increase in difficulty).
- **Balance** — Whether a pool/category gives fair, proportionate coverage to different skills or mechanics, without overrepresenting one.
- **Diversity** — Variation within a pool across an axis (BPM, mapper, song, pattern style) — low diversity is itself a potential finding.
- **Skill Coverage** — Whether the range of skills a tournament should test (streams, jumps, technical, aim, speed, etc.) is actually represented in the pool.
