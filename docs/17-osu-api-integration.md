# osu! API Integration

Undocumented until now: the live osu! API v2 integration that backs Star Rating enrichment, added after [docs/08](08-beatmap-import-pipeline.md) and [docs/10](10-metadata-analyzers.md) were written and originally described Star Rating as fully deferred. Implemented in `backend/internal/osuapi` (the HTTP client), `backend/internal/enrich` (the import-time orchestration), and `frontend/app/api/osu-proxy/route.ts` (a same-origin proxy for beatmap file downloads).

## Why this exists

osu!'s strain-based Star Rating algorithm was never implemented locally (see [docs/08](08-beatmap-import-pipeline.md#what-is-explicitly-deferred)) — it's a substantial standalone computation, and osu! already computes and serves it authoritatively. Rather than build and maintain a second implementation that could drift from osu!'s own values, this project fetches the real value from osu!'s API at import time. This is a deliberate scope choice: `internal/osuapi` and `internal/enrich` are the project's only network-dependent, non-deterministic components — everything else (parsing, normalization, analyzers) stays pure and offline, per Architecture Principle 6 (determinism). Isolating the one I/O-bound exception into its own two packages, rather than threading network calls through `normalize` or an analyzer, keeps that guarantee intact everywhere else.

## `internal/osuapi`: the API client

A narrow client for osu! API v2, covering exactly two operations:

- **`Lookup(ctx, checksum)`** — resolves a beatmap by osu!'s own MD5 checksum via `GET /beatmaps/lookup?checksum=`. Returns `ErrBeatmapNotFound` if osu! has no beatmap with that checksum (e.g. an unranked/unsubmitted map — a normal, expected outcome, not a bug).
- **`StarRating(ctx, beatmapID, mods)`** — fetches Star Rating for a specific beatmap under a specific `modmap.Mods` combination via `POST /beatmaps/{id}/attributes`, computed on demand by osu!'s own difficulty-attributes endpoint rather than limited to a fixed precomputed set.

Both are declared on the `Client` interface, not a concrete type, everywhere they're consumed — `internal/analysis` never imports `osuapi` directly (analyzers stay pure/offline-testable; `tournament.DifficultySpreadAnalyzer` depends on an injected `StarRatingLookup` interface instead, reading already-persisted values — see [docs/12](12-tournament-analyzers.md#difficultyspreadanalyzer-stage-scope)). Only `internal/enrich` and `internal/osuapi`'s own tests ever make a real network call.

### Authentication

OAuth2 client-credentials flow (`oauth.go`'s `tokenSource`), scoped to `public` — no user login, no redirect flow, matching a server-to-server integration rather than a per-user one. The access token is cached in memory and refreshed automatically 30 seconds before expiry (`expiryMargin`). Credentials (`OSU_CLIENT_ID`, `OSU_CLIENT_SECRET`) are read once at startup by `internal/config` (see [docs/18](18-configuration-and-modmap.md)) and never touch the frontend or a client-facing response.

### Error handling

Three typed sentinel errors (`ErrBeatmapNotFound`, `ErrRateLimited`, `ErrUnavailable`) let `internal/enrich` distinguish a permanent "this beatmap has no osu! data" outcome (don't retry) from a transient network/rate-limit blip (skip and log, retry on the next import). The underlying `http.Client` always carries an explicit 10-second timeout — never the zero-value default client — matching `cmd/server/main.go`'s explicit-timeout convention for the HTTP server itself.

## `internal/enrich`: import-time orchestration

`Enricher.Enrich(ctx, beatmap, sourceBytes)` runs once per successfully-imported beatmap, immediately after normalization, and is **best-effort by design**:

1. **Resolve the beatmap's osu! numeric ID.** Prefers the ID already parsed at normalize time (free, no network call) via `domain.Beatmap.OsuBeatmapID`. Falls back to `osuapi.Client.Lookup` against the MD5 checksum of the raw uploaded bytes (osu!'s own checksum algorithm — distinct from this project's sha256 `OsuFileHash` used for import dedup, hence the separate `md5Hex` helper).
2. **Fetch Star Rating for a fixed set of eager mod combinations** (`eagerMods`: NoMod, HardRock, DoubleTime, Easy, HalfTime) — chosen to cover the NM baseline plus every single mod that changes Star Rating under osu!'s classic algorithm, and by extension `modmap.FreeModCandidates` (a subset of `eagerMods`). Hidden and Flashlight are excluded: Hidden never changes Star Rating alone (`modmap.AffectsStarRating`), and multi-mod combos beyond the fixed set (e.g. HDHR) are a documented follow-up, not part of this slice.
3. **Persist each successfully-fetched value** via `storage.StarRatingRepository.Save`, keyed by `(BeatmapID, Mods)`.

An unresolvable beatmap ID (unranked map, or the osu! API unreachable) makes `Enrich` return `nil`, not an error — the caller, `api.ImportBeatmap`, must never fail an import because enrichment couldn't complete; a beatmap with no Star Rating data is a normal, representable state (see `DifficultySpreadAnalyzer`'s `skipped_slots_no_sr_data` metric in [docs/12](12-tournament-analyzers.md)). `Enrich` returns a non-nil error only when an ID *was* resolved but every single mod-combo fetch failed — a distinct, worth-logging "total failure" case.

### Feature gating

Enrichment is entirely optional at runtime. `cmd/server/main.go` only constructs an `enrich.Enricher` and wires it into `api.NewServer` when `config.Config.StarRatingFetchEnabled` is true (both `OSU_CLIENT_ID` and `OSU_CLIENT_SECRET` set); otherwise the server logs that enrichment is disabled and runs with a `nil` `api.Enricher`. `tournament.DifficultySpreadAnalyzer` is registered unconditionally either way — with no Star Rating data available, it degrades gracefully to metrics-only results (`skipped_slots_no_sr_data`), the same "insufficient data" pattern every other analyzer in this codebase follows rather than a separate on/off code path.

## `frontend/app/api/osu-proxy/route.ts`: beatmap file proxy

A same-origin Next.js route handler, unrelated to `internal/osuapi` (it doesn't touch OAuth or Star Rating) but solving an adjacent problem: the pool builder needs to download a beatmap's raw `.osu` file from `https://osu.ppy.sh/osu/{id}` for the frontend's own import flow ([docs/15](15-ui-specification.md#backend-wiring)), and a direct browser-side `fetch` to `osu.ppy.sh` would hit CORS. The proxy re-issues the request server-side (`GET /api/osu-proxy?id={beatmapId}`, a 10-second timeout via `AbortSignal.timeout`) and streams the response back same-origin. It validates `id` is purely numeric before forwarding, and maps upstream failures to `502` rather than passing through osu!'s status verbatim. It carries no osu! API credentials — this is a plain file download, not an authenticated API v2 call, so it does not depend on `internal/config`'s `OsuClientID`/`OsuClientSecret`.

## What is explicitly deferred

- **Multi-mod combo eager-fetching beyond `eagerMods`.** A pool using an uncommon combination (e.g. HDHR, DTHD) will have no persisted Star Rating for that exact combo even though `osuapi.Client.StarRating` could fetch it on demand — `DifficultySpreadAnalyzer` only reads what `Enrich` already persisted, it never calls `osuapi` itself. Fetching on demand for arbitrary combos is a reasonable follow-up once a real usage pattern justifies the added request volume.
- **Retry/backoff on `ErrRateLimited`.** The typed error exists so a caller *could* retry, but `Enrich` currently treats a rate-limited mod-combo fetch the same as any other failure (logged, counted toward the total-failure threshold) rather than retrying with backoff.
- **User-facing OAuth.** This integration is server-to-server only (client-credentials grant); there is no "connect your osu! account" flow anywhere in the product, and none is planned — the project analyzes pools, not player accounts, per [docs/01-vision.md](01-vision.md).

## Testing

- `backend/internal/osuapi/client_test.go` — OAuth token fetch and reuse (cached until near expiry), Star Rating fetch by beatmap+mods, checksum lookup, and error-status-to-typed-error mapping, all against an `httptest.Server` (no real network call).
- `backend/internal/osuapi/osuapitest` — a `FakeClient` test double implementing `osuapi.Client` in-memory, with no `_test.go` of its own since it *is* the test double used by `internal/enrich`'s tests.
- `backend/internal/enrich/starrating_test.go` — ID resolution (parsed ID vs. checksum-lookup fallback), eager mod-combo fetch and persistence, partial-failure tolerance (some mod combos fail, `Enrich` still returns `nil`), and the total-failure error path.

```sh
cd backend && go test ./...
```
