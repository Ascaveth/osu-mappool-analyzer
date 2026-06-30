# Scope

## In scope (current phase of the project)

| Category | Examples |
|---|---|
| Tournament configuration | Arbitrary stages, mod categories, slot counts, ordering, rules — fully user-defined, never hardcoded |
| Beatmap import | Parsing `.osu` files: metadata, timing points, hit objects |
| Metadata analysis | Star rating, BPM, AR/OD/CS/HP, drain time, total length, object count, slider ratio, mapper, artist, genre |
| Pattern analysis | Jump distance/angles/spacing, stream/burst detection, rhythm complexity, slider complexity, spinner usage, object density, flow/precision |
| Tournament-level analysis | Composition, progression, balance, diversity, validation |
| Reporting | Human-readable findings, warnings, recommendations, statistics |
| API | REST endpoints exposing structured analysis results |
| Frontend | Visualization of analysis results — secondary to the engine |
| Tournament comparison | Comparing pools across tournaments or editions |

## Out of scope (current phase)

These are explicitly excluded so the project doesn't drift into adjacent but different problems:

- **Player analysis** — individual player skill, performance, or stats.
- **Team analysis** — team composition, performance, or strategy.
- **Match prediction / score prediction** — anything that forecasts outcomes.
- **AI coaching** — gameplay advice for players.
- **Tournament management** — registration, brackets, staffing.
- **Match scheduling** — calendars, timeslots, logistics.

If a feature request falls into one of these categories, it does not belong in this project, regardless of how related it feels.

## Boundary cases

Some ideas sit close to the line and are worth naming explicitly:

- **Replay analysis** is out of scope now (it concerns player performance), but is listed as a *future idea* if it's ever repurposed toward map-design insight (e.g., aggregate replay data revealing a map's actual difficulty vs. its nominal star rating). Not committed.
- **Historical tournament trends** are in scope as a comparison feature (pool-to-pool), not a player-performance feature.

## Success metrics

The project succeeds if:

1. A mappooler can submit a tournament configuration + beatmap set and receive analysis without manually computing any metric by hand.
2. The Validation Engine catches at least the categories of issues named in the plan (difficulty spikes, repeated styles, weak progression, missing skill coverage) on real, previously-released pools, validated against known community feedback on those pools.
3. New analyzers can be added without modifying existing analyzer code (plugin architecture holds under real use, not just in theory).
4. Reports are written in language a non-technical tournament host can act on without needing to interpret raw numbers.
