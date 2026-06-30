# UI Specification

Phase 11 deliverable: the visual identity and flagship view for presenting `Report` output (`docs/13-report-specification.md`) to a tournament organizer. Implemented in `frontend/` (Next.js + TypeScript + TailwindCSS, per `docs/05-stack-proposal.md`).

Per `pool-lab-plan.md`: "Visualization is secondary to analysis." This phase designs and builds one view in full — the Tournament Report — rather than a shallow pass across every roadmap-listed view (Dashboard, Stage, Category, Beatmap, Comparison, Validation, Reports). That scoping choice is explained at the end of this document.

## The brief, pinned down

**Subject:** osu! tournament mappooling — the practice of assembling a structured set of beatmaps (a "pool") into mod categories (NM/HD/HR/DT/FM/TB) across elimination stages (Qualifiers → Round of 16 → ... → Grand Finals), done today almost entirely in spreadsheets.

**Audience:** tournament organizers and mappoolers — people fluent in the domain's own vocabulary (AR/OD/CS/HP, BPM, slot codes like `NM1`/`HD2`) who are evaluating a pool's quality, not casual visitors.

**The page's one job:** make the Analysis Engine's conclusions about a tournament's pool legible at a glance, in the pool's own structure — not a generic analytics dashboard bolted onto unrelated data.

## Design plan

### Color

| Token | Hex | Role |
|---|---|---|
| `paper` | `#E9EDE1` | Page background — ledger paper, not the cliché warm cream |
| `paper-line` | `#C7D0B5` | Hairlines, row dividers, banding |
| `ink` | `#1B1F17` | Primary text |
| `ink-soft` | `#5C6450` | Secondary text — captions, artist credits, reasons |
| `mark` | `#8C2F2F` | Annotation ink — finding glyphs and emphasis, oxblood |
| `brass` | `#B5742A` | The one interactive/accent color — slot codes, stage numerals, focus ring, selection |

Six colors, four of which carry a specific job (paper, line, ink, ink-soft) and two of which are reserved for meaning (mark = "the engine flagged this," brass = "you can act on this"). Deliberately not the AI-default cream-serif-terracotta combination, not nameless near-black, and not a literal newspaper — it's ledger paper, picked because the subject is a sheet of structured records being audited line by line, the same way an accountant's ledger is.

### Type

- **Display — Fraunces** (italic, restrained use): the Report's narrative `summary` sentence, stage names, the masthead title. Fraunces' soft, slightly old-style serif reads as "this was composed for you to read," not "this is a UI label."
- **Body — IBM Plex Sans**: beatmap titles, marginalia note prose.
- **Data — IBM Plex Mono**: slot codes (`NM1`), AR/OD/BPM readouts, statistics line, eyebrow labels, timestamps. Plex Sans and Plex Mono are siblings in the same type superfamily — built for technical/engineering contexts — which is exactly what an "Analysis Engine" is.

### Layout concept

```
┌─────────────────────────────────────────┐
│ MAPPOOL ANALYSIS · TOURNAMENT REPORT     │  ← eyebrow, mono
│ Spring Invitational 2026                 │  ← masthead, Fraunces
├─────────────────────────────────────────┤
│ "Difficulty cools off right when         │  ← thesis: the Report's
│  Round of 16 should be raising..."       │    own summary sentence,
│                                           │    set as the hero
│ 12 analyses · 4 findings · 4 warnings    │  ← quiet statistics line
├─────────────────────────────────────────┤
│ I  Qualifiers              ┊             │  ← stage, roman numeral
│   NM  NM1 Glasswing  AR.. ┊             │
│       NM2 Tidesong   AR.. ┊             │
│   HD  HD1 ...              ┊             │
├─────────────────────────────────────────┤
│ II Round of 16             ┊ ▲ one      │  ← marginalia: findings
│   NM  NM1 ...               ┊ category   │    annotate the ledger,
│       ...                   ┊ holds 75%  │    not a separate panel
└─────────────────────────────────────────┘
```

### Signature: marginalia, not a findings panel

The product's own architecture already decided this: a Report is "narrative sections... that cite their source Analyses, not duplicate them" (`docs/06-domain-model.md`), and Architecture Principle 9 requires findings to read as conclusions, explaining *why*, not as a number on a chart. A findings *panel* — a list of cards under the pool — would separate the conclusion from the thing it's about. Instead, every Finding renders as a **marginal annotation**, in a right-hand gutter beside the exact stage or category it concerns, the way an editor annotates a manuscript: a small mark (▲ warning, ● critical, · info), the finding's description, its reason, and which analyzer raised it — visually attached to the row it's about via grid placement, not a separate "Findings" section a reader has to cross-reference back to the pool.

This is also why the Report's `summary` is the page's hero, not a stat block: the product's own philosophy says a report should open with a conclusion in prose (`docs/04-architecture-principles.md` Principle 9's own example — "the Finals stage introduces a noticeable increase in technical difficulty" — is exactly this kind of sentence). The hero is literally that sentence, set large in italic Fraunces, with the numeric statistics relegated to a quiet mono line beneath it.

### Why roman numerals, here

Numbered/lettered markers are a generic AI-design tell when they decorate non-sequential content. They're used here because `Stage.Order` is "explicit and authoritative for progression analysis — stage sequence is never inferred from name or slice position" (`docs/06-domain-model.md`) — the order is a real, load-bearing fact about the domain, not decoration. Roman numerals (I, II, III) were chosen over `01/02/03` specifically to read as "movements in a programme" rather than the generic numbered-card pattern, reinforcing the editorial/programme framing rather than a dashboard's step indicator.

### Motion

A single, restrained page-load sequence: the masthead, thesis, and each stage section rise in with a short, staggered delay (~90ms per stage) — evoking the engine "typing down the page" as it finishes each section, not a flashy entrance. No looping or ambient animation. `prefers-reduced-motion: reduce` disables it entirely.

## What was built

- `frontend/` — Next.js 16 (App Router) + TypeScript + Tailwind v4, scaffolded fresh for this phase.
- `app/globals.css` — the full token system and bespoke layout CSS (`.programme`, `.stage`, `.marginalia`, etc.) described above.
- `components/Masthead.tsx`, `ThesisHero.tsx`, `StageSection.tsx`, `MarginNote.tsx` — the four components composing the flagship view.
- `lib/types.ts` — presentation-layer types shaped after `docs/api/openapi.yaml`'s schemas (camelCased, no new fields).
- `lib/sample-data.ts` — a hand-built sample `Tournament` and `Report` exercising all four Phase 8 analyzers (composition, progression, balance, diversity) with realistic, deliberately-flawed pool data, since no live backend is wired up yet (Phase 10 delivered the API *contract*, not a running server — see `docs/14-api-specification.md`'s scope note).

No shadcn/ui components were pulled in for this view — its bespoke ledger/marginalia grid doesn't fit shadcn's card-and-panel defaults, and using a generic kit here would have worked against the brief's whole point (a visual identity that couldn't be mistaken for any other tool). shadcn remains the right call for future *utility* screens (a tournament-configuration form, import flow) where consistency matters more than novelty — see Future Work below.

Verified: `npx tsc --noEmit` (clean), `npm run lint` (clean), `npm run build` (static prerender succeeds), manual screenshot review at desktop (1280px) and mobile (390px) widths, zero browser console errors.

## What is explicitly not in scope for this phase

Matching Phase 10's precedent of scoping documentation-and-design phases honestly rather than half-building every roadmap item:

- **Other views** (Dashboard, standalone Stage/Category/Beatmap pages, Comparison, Validation). The roadmap lists these, but building seven shallow views would dilute the one thing this phase needed to prove: a visual identity distinctive enough that "give every client a look that couldn't be mistaken for anyone else's" is actually true. The token system and components here (`Masthead`, `ThesisHero`, `StageSection`, `MarginNote`) are the reusable basis a Dashboard or Beatmap detail view would build from next, not a one-off.
  - **Comparison** and **Validation** specifically have no view because they have no backing endpoint yet (`docs/14-api-specification.md`'s "why there is no `/validations` or `/comparisons` endpoint" applies identically here — there's nothing for a Validation or Comparison page to render that the Report view doesn't already show via marginalia/`severity`).
- **Live data.** The view renders `lib/sample-data.ts`, not a fetch against the Phase 10 API — there is no running backend server yet (`docs/14-api-specification.md`'s scope note). Wiring a real `fetch`/`getServerSideProps`-equivalent against `GET /tournaments/{id}/report` is a follow-up once a server implementation exists.
- **shadcn/ui integration**, for the reason above — deferred to whichever future screen is utility-first rather than identity-first.
