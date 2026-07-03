# Process Configuration and Mod-Category Mapping

Undocumented until now: two small, unrelated backend modules that support the [osu! API integration](17-osu-api-integration.md) — `backend/internal/config` (process configuration) and `backend/internal/modmap` (osu! mod ↔ tournament category translation). Neither is a domain concept in its own right; both exist to let other packages stay decoupled from environment/convention details.

## `internal/config`: process configuration

Loads process-wide settings once at startup from environment variables (`Load()`, called once by `backend/cmd/server/main.go`). Two categories of setting, handled differently on purpose:

- **`Port`, `AllowedOrigins`** — optional tunables with safe defaults (`PORT` defaults to `"8080"`; `ALLOWED_ORIGINS` is a comma-separated list, defaulting to `http://localhost:3000` for local frontend development). Absence is a normal, unremarkable state.
- **`OsuClientID`, `OsuClientSecret`** — required-together secrets gating [osu! API Star Rating enrichment](17-osu-api-integration.md#internal-osuapi-the-api-client). `Load()` fails loudly (returns an error, which `main.go` treats as fatal at startup) only when exactly one of the pair is set — almost certainly a deployment mistake — but treats *both* being empty as a legitimate "feature not in use" state, not an error. `Config.StarRatingFetchEnabled` is `true` only when both are present, and is the single flag `main.go` checks to decide whether to construct an `enrich.Enricher` at all.

`godotenv.Load()` in `main.go` loads a `backend/.env` file into the process environment before `config.Load()` runs, for local development convenience; a missing `.env` file is not an error (real deployments set env vars directly), but a malformed one that does exist is fatal.

## `internal/modmap`: mod-category translation

Translates this project's free-text `Category.Name` convention (`"NM"`, `"HD"`, `"HR"`, `"DT"`, `"FM"`, `"TB"`, or a combo like `"HDHR"` — see [docs/07](07-tournament-configuration.md#configuration-shape)) into osu! mod bitflags (`modmap.Mods`), the shape the osu! API needs to fetch mod-specific Star Rating. This is deliberately a **named, overridable convention table**, not a domain rule enforced by `internal/domain` — [Architecture Principle 4](04-architecture-principles.md#4-tournament-structure-is-always-user-defined) means `Category.Name` is never validated against a fixed enum at the domain layer, and `modmap` doesn't change that; it's a best-effort mapping one specific consumer (Star Rating enrichment and `DifficultySpreadAnalyzer`) opts into, the same way `tournament.DefaultTaxonomy()` keeps its skillset conventions outside `internal/domain` too.

`Mods` is defined in `modmap`, not in `internal/osuapi`, specifically so `internal/analysis/tournament` can depend on it without pulling in the I/O-concerned `osuapi` package — `osuapi` depends on `modmap`, never the reverse.

### What it does

- **`FromCategoryName(name string) (Mods, bool)`** — resolves a category name to the `Mods` that change Star Rating for it. `"NM"` maps to `NoMod` (a resolvable, explicit zero value); `"FM"` and `"TB"` return `(0, false)` — "no single fixed mod," not NoMod — since a FreeMod or Tiebreaker slot's actual mods aren't fixed at pool-build time. Combo names (`"HDHR"`, `"DTHD"`) are decomposed two characters at a time and OR'd together; an unrecognized token, or an odd-length name, makes the whole name unresolvable.
- **`IsFreeMod(name string) bool`** — distinguishes the FreeMod convention specifically from other unresolvable names (typos, `"TB"`), so a caller can apply `FreeModCandidates` ranging instead of skipping the slot outright.
- **`FreeModCandidates`** — `{NoMod, HardRock, Easy}`, the mods a FreeMod slot's difficulty *range* is computed across. Deliberately excludes DoubleTime, even though DT changes Star Rating the same way HR does — DT is not a legal FreeMod pick under this project's tournament convention.
- **`AffectsStarRating(m Mods) bool`** — reports whether `m` includes any mod that actually changes Star Rating under osu!'s classic (stable) algorithm. Hidden and Flashlight are tracked in the `Mods` bitflag (so combo names like `"HDHR"` still decompose correctly) but never affect this check on their own — only HardRock, DoubleTime, Easy, and HalfTime do.

### Where it's used

- **`internal/enrich`**: `eagerMods`, the fixed set of mod combinations fetched per beatmap at import time, is chosen to be a superset of `modmap.FreeModCandidates` — see [docs/17](17-osu-api-integration.md#internal-enrich-import-time-orchestration).
- **`internal/osuapi`**: `modsToAPINames` converts a `modmap.Mods` bitflag set into the acronym list (`"HR"`, `"DT"`, etc.) the osu! API's attributes endpoint expects.
- **`tournament.DifficultySpreadAnalyzer`** ([docs/12](12-tournament-analyzers.md#difficultyspreadanalyzer-stage-scope)): uses `FromCategoryName` to resolve each slot's fixed mod (or lack thereof), `IsFreeMod`/`FreeModCandidates` to range FreeMod slots instead of skipping them, and reports `skipped_slots_no_fixed_mod` for any category (like `"TB"`) with no fixed-mod convention at all.

## Testing

- `backend/internal/config/config_test.go` — table-driven, covering both env vars unset, both set, one-set/one-missing (the error case), and the `PORT`/`ALLOWED_ORIGINS` defaults.
- `backend/internal/modmap/modmap_test.go` — `FromCategoryName` single-name and combo resolution, unresolvable-name handling, `IsFreeMod`, `FreeModCandidates` excluding DoubleTime, and `AffectsStarRating`.

```sh
cd backend && go test ./...
```
