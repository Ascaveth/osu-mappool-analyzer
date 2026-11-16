# osu! Mappool Analyzer — Report Viewer

The frontend for osu! Mappool Analyzer's flagship view: a tournament report that
presents the Analysis Engine's findings (composition, progression, balance, and
diversity) as a readable, citation-annotated report for tournament organizers
and mappoolers — stage by stage, category by category, beatmap by beatmap.

Set `NEXT_PUBLIC_API_BASE_URL` to connect to the Go backend. Without it, the
frontend uses a localStorage-backed demo client. See `docs/15-ui-specification.md` in the
repository root for the full UI specification and scope notes.

## Stack

- [Next.js](https://nextjs.org) (App Router) + React + TypeScript
- Tailwind CSS v4 + daisyUI v5 for buttons, inputs, alerts, loading, and toggles
- Fonts: [Fraunces](https://fonts.google.com/specimen/Fraunces) (display serif),
  IBM Plex Sans (body), IBM Plex Mono (data/labels), loaded via `next/font`

## Getting started

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) to view the report.

- `app/page.tsx` — landing page and entry into tournament setup
- `app/tournaments/` — tournament setup, pool import, and report views
- `components/` — `Masthead`, `ThesisHero`, `StageSection`, `MarginNote`
- `lib/types.ts` — presentation-layer types mirroring the backend's API shape
- `lib/api/` — REST client and localStorage-backed demo client

## Other commands

```bash
npm run build   # production build
npm run start   # serve the production build
npm run lint    # eslint
```

## Interface checks

```bash
npx playwright install chromium
npm run test:ui
```

The tests build and serve the production frontend on port 3100 in demo mode.
Beatmap provider responses are intercepted; no backend or osu! access is needed.
They cover narrow layouts, import errors and focus restoration, bulk import,
report explanations, primary-button contrast, and reduced-motion theme changes.
Screenshots are written under the ignored `test-results/` directory.

`next-themes` owns the saved preference and writes `data-theme` on the root.
`app/globals.css` maps daisyUI colors onto the Programme palette. The report
layout and typography remain custom. Keep component state styling in daisyUI;
local overrides should cover editorial typography and domain-specific layout.
