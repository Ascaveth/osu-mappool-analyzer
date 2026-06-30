# osu! Mappool Analyzer — Report Viewer

The frontend for osu! Mappool Analyzer's flagship view: a tournament report that
presents the Analysis Engine's findings (composition, progression, balance, and
diversity) as a readable, citation-annotated report for tournament organizers
and mappoolers — stage by stage, category by category, beatmap by beatmap.

This view currently renders against sample data (`lib/sample-data.ts`); it is
not yet wired to a live backend. See `docs/15-ui-specification.md` in the
repository root for the full UI specification and scope notes.

## Stack

- [Next.js](https://nextjs.org) (App Router) + React + TypeScript
- Tailwind CSS v4
- Fonts: [Fraunces](https://fonts.google.com/specimen/Fraunces) (display serif),
  IBM Plex Sans (body), IBM Plex Mono (data/labels), loaded via `next/font`

## Getting started

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) to view the report.

- `app/page.tsx` — assembles the report from `lib/sample-data.ts`
- `components/` — `Masthead`, `ThesisHero`, `StageSection`, `MarginNote`
- `lib/types.ts` — presentation-layer types mirroring the backend's API shape
- `lib/sample-data.ts` — mock tournament + report data for this view

## Other commands

```bash
npm run build   # production build
npm run start   # serve the production build
npm run lint    # eslint
```
