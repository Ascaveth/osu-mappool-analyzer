# osu! Mappool Analyzer

Analysis tool for osu! tournament mappools. You feed it a tournament's stages and beatmaps; it runs a set of analyzers over the pool and reports back on composition, progression, balance, and diversity, the things a mappooler actually needs to sanity-check before a pool goes live.

This isn't a player stats site, and it's not another leaderboard clone. The pool itself is the subject.

## Where things stand

Two pieces, wired together:

- **Backend** (`backend/`): a Go API with a working analysis engine behind it. Tournaments, stages, categories, slots, beatmaps — all modeled and exposed over REST.
- **Frontend** (`frontend/`): a Next.js app. The tournament creation flow, pool editor, and report view (`app/tournaments/new`, `app/tournaments/[id]/pool`, `app/tournaments/[id]/report`) call the backend through a real REST client in `lib/api/`. The root page (`app/page.tsx`) is a standalone demo that still renders off `lib/sample-data.ts`.

Storage is in-memory only. Restart the backend and every tournament you created is gone; there's no Postgres or any other persistence layer plugged in yet, despite that being the long-term plan.

## What the analyzers actually do

Under `backend/internal/analysis/`:

**Tournament-level** (`tournament/`): balance, composition, diversity, progression. These look at a pool across its whole stage list: is Round of 16 meaningfully harder than Qualifiers, does one mod category dominate, that sort of question.

**Metadata-level** (`metadata/`): BPM range, difficulty settings, mapper repetition, object density. Per-beatmap and per-slot checks that feed into the tournament-level analyzers above.

**Pattern-level** (`pattern/`): jump distance, jump angle, slider complexity, spinner usage, stream bursts. Lower-level readouts of what's actually in the beatmap file, parsed by `internal/osufile`.

None of this is hardcoded to a fixed bracket shape. Tournament structure — stages, slot counts, mod categories — is user-defined, because no two tournaments run the same format.

## Architecture

Go backend, stdlib HTTP plus `google/uuid` (that's the only external dependency — no framework). It exposes a REST API over the analysis engine and normalizes raw beatmap data before anything touches an analyzer. The Next.js frontend is a presentation layer on top; it doesn't do analysis itself.

Full design docs live in `docs/` (`01-vision.md` through `16-test-plan.md`). Treat those as specification, not a changelog of what's built — plenty of it is still ahead of the code.

## Running it

With Docker:

```bash
docker-compose up
```

Backend on `localhost:8080`, frontend on `localhost:3000`. The frontend build bakes in `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/v1` at build time since the API calls happen client-side, not from Next's server.

Without Docker, run each side manually:

```bash
# backend
cd backend && go run ./cmd/server

# frontend
cd frontend && npm install && npm run dev
```

## Layout

```
backend/
  cmd/server          entrypoint
  internal/api        REST handlers, routing, pagination, error responses
  internal/analysis   the engine: tournament/, metadata/, pattern/ analyzers
  internal/normalize   normalization pipeline
  internal/osufile     osu! beatmap file parsing
  internal/storage     repository interfaces + in-memory impl

frontend/
  app/tournaments      new-tournament flow, pool editor ([id]/pool), report view ([id]/report)
  components/          report UI (Masthead, ThesisHero, StageSection, MarginNote)
  lib/api/              REST client that talks to the Go backend
  lib/sample-data.ts    mock data used only by the root demo page
```

## Testing

```bash
cd backend && go test ./...
```

That's what CI runs (`.github/workflows/backend-tests.yml`). The frontend has no test suite yet — `npm run lint` is the only check configured there.

## License

MIT, see `LICENSE`.
