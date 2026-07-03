# UI Specification

Phase 11 deliverable, since extended past its original scope: the visual identity and the full workflow of screens for producing and presenting `Report` output (`docs/13-report-specification.md`) to a tournament organizer. Implemented in `frontend/` (Next.js + TypeScript + TailwindCSS, per `docs/05-stack-proposal.md`).

Per `pool-lab-plan.md`: "Visualization is secondary to analysis." Phase 11 originally designed and built one view in full — the Tournament Report — rather than a shallow pass across every roadmap-listed view. Since then the frontend has grown the utility screens that view depends on: a tournament-configuration wizard and a pool builder/import screen, plus the landing page's own onboarding content. This document has been updated to describe that whole flow as shipped, not just the original flagship view. The rest of the roadmap's views (standalone Dashboard, Comparison, Validation) remain out of scope — see the closing section.

## The brief, pinned down

**Subject:** osu! tournament mappooling — the practice of assembling a structured set of beatmaps (a "pool") into mod categories (NM/HD/HR/DT/FM/TB) across elimination stages (Qualifiers → Round of 16 → ... → Grand Finals), done today almost entirely in spreadsheets.

**Audience:** tournament organizers and mappoolers — people fluent in the domain's own vocabulary (AR/OD/CS/HP, BPM, slot codes like `NM1`/`HD2`) who are evaluating a pool's quality, not casual visitors.

**The product's job, end to end:** let an organizer describe their tournament's structure, fill it with real beatmaps, and get the Analysis Engine's conclusions back legible at a glance, in the pool's own structure — not a generic analytics dashboard bolted onto unrelated data.

## Page inventory

The app is a four-page linear flow plus a landing page, all under `frontend/app/`:

| Route | File | Purpose |
|---|---|---|
| `/` | `app/page.tsx` | Landing page — masthead, a one-line hook, a 4-step "How to use this analyzer" guide (`components/HowToUse.tsx`), a WIP/alpha disclaimer (`components/WipDisclaimer.tsx`), and the entry CTA into the wizard. |
| `/tournaments/new` | `app/tournaments/new/page.tsx` | Tournament-creation wizard ("Step 1 of 2"). Organizer names the tournament and builds an arbitrary list of stages, each with its own mod categories and slot counts, plus an optional projected Star Rating per stage. A live "Checks before you continue" side panel validates the draft (missing names, duplicate mod categories in one stage, malformed Star Rating) before enabling submission. On submit, calls `api.createTournament` and routes to the pool builder. |
| `/tournaments/[id]/pool` | `app/tournaments/[id]/pool/page.tsx` | Pool builder ("Step 2 of 2"). Renders every stage/category/slot generated from the wizard's structure. Each empty slot takes a pasted beatmap URL or ID; per-slot "Import & assign" calls `api.importBeatmapFromUrl` then `api.assignBeatmap`. A bulk "Import All" action drains every pasted-but-unassigned slot with bounded concurrency (3 in flight) and a single refresh at the end. "Run Analysis" is disabled until every slot in the pool is filled, then routes to the report. |
| `/tournaments/[id]/report` | `app/tournaments/[id]/report/page.tsx` | The flagship Report view (unchanged design language from the original Phase 11 spec — see below). Fetches both the `Tournament` and its `Report` live, then renders the masthead, thesis hero, sticky stage nav, and per-stage marginalia findings. |

This replaces the earlier "one flagship Report view" framing: the Report page is still the most bespoke and highest-effort screen, but it is now the third step of a real wizard→builder→report flow, not a standalone view fed by hand-authored fixtures.

## Backend wiring

Live. `frontend/lib/api/rest.ts` implements the shared `ApiClient` interface (`frontend/lib/api/client.ts`) against the Go server's REST API (`docs/api/openapi.yaml`), translating the wire `snake_case` shape into `lib/types.ts`'s camelCase presentation types at the boundary — the frontend mirror of the role `internal/normalize` plays for `.osu` syntax on the backend. It covers tournament creation, tournament fetch (with N+1-avoiding batched beatmap resolution), beatmap import (via a same-origin `/api/osu-proxy` route to sidestep CORS/auth on `osu.ppy.sh`), slot assignment/clearing, and report generation.

Client selection (`frontend/lib/api/index.ts`) is an environment switch: if `NEXT_PUBLIC_API_BASE_URL` is set, `createRestClient` is used; otherwise the app falls back to `createMockClient` (`frontend/lib/api/mock.ts`), a localStorage-backed mock implementing the same `ApiClient` interface, so the app can still demo standalone with no server running. This mock is a genuine runtime fallback, not test scaffolding.

`frontend/lib/sample-data.ts` (the hand-built fixture `Tournament`/`Report` from the original Phase 11 build) still exists in the repo but is no longer imported anywhere in `app/` — it's dead code left over from before live wiring landed and is a candidate for deletion in a future cleanup pass rather than a real fallback path.

## shadcn/ui integration

Corrected from the original spec: shadcn-style components (Radix UI primitives + Tailwind, per `docs/05-stack-proposal.md`) are now in the codebase under `frontend/components/ui/`:

- `tabs.tsx` (`@radix-ui/react-tabs`)
- `switch.tsx` (`@radix-ui/react-switch`)
- `radio-group.tsx` (`@radix-ui/react-radio-group`)
- `dropdown-menu.tsx` (`@radix-ui/react-dropdown-menu`)
- `theme-switch.tsx` — light/dark mode toggle, wired into the shared layout
- `demo.tsx` — a scratch/reference file exercising the above, not a shipped page

The bespoke, identity-first Report view (masthead, thesis hero, stage sections, marginalia) still does not use shadcn — that design intentionally doesn't fit shadcn's card-and-panel defaults, and the reasoning in the original spec (a visual identity that couldn't be mistaken for any other tool) still holds there. shadcn was pulled in for exactly the case the original spec predicted: utility surfaces (theme switching, and available for future form controls) where consistency matters more than novelty, plus the CSS-variable bridge in `globals.css` (`--color-primary`, `--color-border`, `--color-ring`, etc., all mapped onto the ledger-paper token set) so any shadcn component dropped in later automatically inherits the same palette instead of shadcn's own defaults.

## Design plan

### Color

| Token | Hex (light) | Hex (dark) | Role |
|---|---|---|---|
| `paper` | `#E9EDE1` | `#1B1D17` | Page background — ledger paper, not the cliché warm cream |
| `paper-line` | `#C7D0B5` | `#3A3F30` | Hairlines, row dividers, banding |
| `ink` | `#1B1F17` | `#E9EDE1` | Primary text |
| `ink-soft` | `#5C6450` | `#A3AB92` | Secondary text — captions, artist credits, reasons |
| `mark` | `#8C2F2F` | `#D9776A` | Annotation ink — finding glyphs and emphasis, oxblood |
| `brass` | `#B5742A` | `#D69A52` | The one interactive/accent color — slot codes, stage numerals, focus ring, selection |

Still six colors doing the same jobs the original spec described (paper, line, ink, ink-soft for structure; mark = "the engine flagged this," brass = "you can act on this"). The set now has a verified dark-mode pairing (`app/globals.css`'s `:root`/`.dark` blocks), toggled via `components/ui/theme-switch.tsx`, that preserves the same relative contrast and role assignments rather than inverting to a generic dark theme. Deliberately still not the AI-default cream-serif-terracotta combination, not nameless near-black, and not a literal newspaper — it's ledger paper, picked because the subject is a sheet of structured records being audited line by line, the same way an accountant's ledger is.

Mod categories (NM/HD/HR/DT/FM/TB) additionally get their own desaturated accent tokens (`--mod-nm`, `--mod-hd`, etc., `--mod-tb` reusing `--brass`) layered on top of this palette, used for the small category dots in the wizard and pool builder and for slot-row accent styling once a beatmap has cover art (`frontend/lib/beatmap-format.ts`'s `modAccentColor`/`slotAccentStyle`).

### Type

- **Display — Fraunces** (italic, restrained use): the Report's narrative `summary` sentence, stage names, masthead titles across all pages. Fraunces' soft, slightly old-style serif reads as "this was composed for you to read," not "this is a UI label."
- **Body — IBM Plex Sans**: beatmap titles, marginalia note prose, form labels and body copy in the wizard/pool builder.
- **Data — IBM Plex Mono**: slot codes (`NM1`), AR/OD/BPM readouts, statistics lines, eyebrow labels, timestamps, step indicators ("Step 1 of 2"). Plex Sans and Plex Mono are siblings in the same type superfamily — built for technical/engineering contexts — which is exactly what an "Analysis Engine" is.

This type system is unchanged from the original spec and is now applied consistently across all four pages, not just the Report.

### Layout concept (Report page)

```
┌─────────────────────────────────────────┐
│ MAPPOOL ANALYSIS · TOURNAMENT REPORT     │  ← eyebrow, mono
│ Spring Invitational 2026                 │  ← masthead, Fraunces
├─────────────────────────────────────────┤
│ [Qualifiers] [Round of 16] [Finals]      │  ← sticky stage nav
│                                           │    (3+ stages only)
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

The wizard and pool builder pages reuse the same masthead/eyebrow/mono-step-indicator language but trade the marginalia grid for, respectively, a two-column form-plus-checks layout and a flat per-stage slot list — see "Upstream screens" below.

### Signature: marginalia, not a findings panel

The product's own architecture already decided this: a Report is "narrative sections... that cite their source Analyses, not duplicate them" (`docs/06-domain-model.md`), and Architecture Principle 9 requires findings to read as conclusions, explaining *why*, not as a number on a chart. A findings *panel* — a list of cards under the pool — would separate the conclusion from the thing it's about. Instead, every Finding renders as a **marginal annotation** (`components/MarginNote.tsx`), in a right-hand gutter beside the exact stage or category it concerns, the way an editor annotates a manuscript: a small mark (▲ warning, ● critical, · info), the finding's description, its reason, and which analyzer raised it — visually attached to the row it's about via grid placement, not a separate "Findings" section a reader has to cross-reference back to the pool.

This is also why the Report's `summary` is the page's hero, not a stat block: the product's own philosophy says a report should open with a conclusion in prose (`docs/04-architecture-principles.md` Principle 9's own example — "the Finals stage introduces a noticeable increase in technical difficulty" — is exactly this kind of sentence). The hero is literally that sentence, set large in italic Fraunces, with the numeric statistics relegated to a quiet mono line beneath it.

### Why roman numerals, here

Numbered/lettered markers are a generic AI-design tell when they decorate non-sequential content. They're used here because `Stage.Order` is "explicit and authoritative for progression analysis — stage sequence is never inferred from name or slice position" (`docs/06-domain-model.md`) — the order is a real, load-bearing fact about the domain, not decoration. Roman numerals (I, II, III) were chosen over `01/02/03` specifically to read as "movements in a programme" rather than the generic numbered-card pattern, reinforcing the editorial/programme framing rather than a dashboard's step indicator.

### Motion

A single, restrained page-load sequence: the masthead, thesis, and each stage section rise in with a short, staggered delay (~90ms per stage) — evoking the engine "typing down the page" as it finishes each section, not a flashy entrance. No looping or ambient animation. `prefers-reduced-motion: reduce` disables it entirely. The landing page's `HowToUse` steps use the same restrained `reveal` fade-in treatment on load.

## Upstream screens: wizard and pool builder

These two screens didn't exist in the original Phase 11 spec and are documented here for the first time.

**Tournament wizard (`/tournaments/new`).** A two-column layout: the left column is the editable draft (tournament name, then a repeatable list of stages, each a repeatable list of mod categories with slot counts and an optional per-stage projected Star Rating); the right column is a persistent "Checks before you continue" panel that mirrors the marginalia idea from the Report — every validation problem (missing name, duplicate mod category within a stage, malformed Star Rating) renders as its own annotated note with a warning or ready mark, live, rather than being deferred to a single post-submit error. This keeps the same "the ledger tells you what's wrong, in place" philosophy the Report established, applied to data entry instead of analysis output.

**Pool builder (`/tournaments/[id]/pool`).** A flat, per-stage/per-category/per-slot list generated directly from the structure the wizard just created — no separate "add a slot" step, because slot count was already fixed at tournament-creation time (consistent with `docs/06-domain-model.md`'s treatment of pool structure as tournament configuration, not freeform data entry). Each unfilled slot is a paste-URL-and-confirm row; filled slots collapse into a compact `slot-chip` showing the beatmap's title/artist and AR/OD/BPM readout, using cover art as a background accent when available. "Run Analysis" is gated on `filledCount === totalCount`, preventing a report from ever being generated against an incomplete pool.

## Recent UX polish

A set of small but user-facing improvements landed after the original Report view shipped, spanning all three interactive pages:

- **Import loading states.** Per-slot imports show an inline spinner (`.spinner`) in place of the confirm checkmark while `api.importBeatmapFromUrl` + `api.assignBeatmap` are in flight, and the "Import All" bulk action shows the same spinner with an in-progress label ("Importing…"). Both actions are disabled while a request is outstanding to prevent duplicate submissions.
- **Dimmed/blurred overlay during import.** While any beatmap import is running (single-slot or bulk), a full-page `.loading-overlay` (`role="status"`, `aria-live="polite"`) dims and blurs the pool builder behind a large spinner and a status line ("Importing N beatmaps…" / "Importing beatmap…"), making it unambiguous that the page is mid-operation rather than stalled, without blocking the browser or losing scroll position.
- **Bulk paste import.** The pool builder's "Import All" button drains every slot that has a pasted-but-unconfirmed URL, at a bounded concurrency of 3 concurrent imports, then performs a single tournament refresh at the end instead of one refresh per slot — meant for organizers pasting an entire pool's worth of links at once rather than confirming each slot individually.
- **Unified error/severity styling.** A shared `.alert` component (eyebrow-style icon + text, oxblood `--mark` accent) now renders top-level errors consistently across the wizard, pool builder, and report pages, replacing what had been page-specific ad hoc error rendering. Inline per-slot import errors use the same visual language at smaller scale (`.slot-error`).
- **Severity-colored notes and stage nav.** The wizard's "Checks before you continue" notes use the same `.note-mark` severity-coloring vocabulary (`--critical`/`--warning`/`--info`/`--ready`) that the Report's marginalia already used, so "this needs attention" reads identically whether it's a setup validation or an analyzer finding. The Report gained a sticky `StageNav` (`components/StageNav.tsx`) — a row of jump links, one per stage, self-hiding for pools of 2 or fewer stages where scrolling between sections is already trivial.
- **Landing page tutorial and WIP disclaimer.** The landing page (`app/page.tsx`) now carries a 4-step "How to use this analyzer" walkthrough (`components/HowToUse.tsx`) covering setup → import → analyze → read findings, and a standing work-in-progress disclaimer (`components/WipDisclaimer.tsx`) reminding organizers that analyzer output is a set of suggestions to evaluate, not a verdict — surfaced once on the landing page rather than repeated on every report.
- **Favicon.** A properly cropped, multi-resolution `.ico` replaced a placeholder favicon (`fix(frontend): crop favicon tight and encode proper multi-res .ico`) — cosmetic, but part of the same "looks shipped, not scaffolded" polish pass.

## What was built

- `frontend/` — Next.js (App Router) + TypeScript + Tailwind v4.
- `app/globals.css` — the full token system (light and dark), the bespoke layout CSS (`.programme`, `.stage`, `.marginalia`, `.loading-overlay`, `.alert`, `.note-mark`, etc.), and the CSS-variable bridge feeding shadcn/Tailwind's own color tokens.
- `components/Masthead.tsx`, `ThesisHero.tsx`, `StageSection.tsx`, `StageNav.tsx`, `MarginNote.tsx` — the Report view.
- `components/HowToUse.tsx`, `WipDisclaimer.tsx`, `Footer.tsx` — landing-page and sitewide chrome.
- `components/ui/` — shadcn-style primitives (`tabs`, `switch`, `radio-group`, `dropdown-menu`, `theme-switch`) built on Radix UI, styled through the ledger-paper token bridge.
- `lib/types.ts` — presentation-layer types shaped after `docs/api/openapi.yaml`'s schemas (camelCased, no new fields).
- `lib/api/client.ts` — the shared `ApiClient` interface.
- `lib/api/rest.ts` — the live REST client against the Go backend.
- `lib/api/mock.ts` — the localStorage-backed fallback client used when no backend is configured.
- `lib/api/index.ts` — the environment-driven switch between the two.
- `lib/sample-data.ts` — retained from the original Phase 11 build but no longer referenced by any page; a cleanup candidate.
- `app/tournaments/new/page.tsx`, `app/tournaments/[id]/pool/page.tsx` — the wizard and pool builder screens described above.

## What is explicitly not in scope

- **Other roadmap views** (standalone Dashboard, standalone Stage/Category/Beatmap detail pages, Comparison, Validation). The roadmap lists these, but the Report view's marginalia already surfaces severity-scoped findings at the stage/category/beatmap level, and **Comparison** and **Validation** specifically still have no view because they have no backing endpoint (`docs/14-api-specification.md`'s "why there is no `/validations` or `/comparisons` endpoint" still applies).
- **Removing `lib/sample-data.ts`.** Left in place for now since it's inert (unreferenced by any page) rather than actively misleading, but should be deleted in a future `chore(frontend)` pass rather than continuing to imply the app depends on fixture data.
- **Further shadcn adoption.** The primitives currently in `components/ui/` (tabs, switch, radio-group, dropdown-menu, theme switch) are what's shipped; the wizard and pool builder's form controls are still bespoke `.field-input`/`.field-select` markup rather than shadcn `Input`/`Select` equivalents. Migrating them is a reasonable follow-up once there's a second or third form-heavy screen to justify the shared component investment.
