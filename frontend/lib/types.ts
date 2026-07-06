// Presentation-layer types, deliberately shaped like the OpenAPI contract
// (docs/api/openapi.yaml) but camelCased for idiomatic TypeScript — a
// real API client would translate snake_case wire fields into these, the
// same way `normalize.Beatmap` translates `.osu` file syntax into
// `domain.Beatmap` on the backend. No field here exists that the backend
// doesn't already produce; this file does not invent new data shape.
// Exception: `Beatmap.coverUrl` is client-derived from the public osu!
// CDN by beatmapset ID and is never sourced from the backend — it's a
// pure presentation value, not part of the domain model.

export type ScopeType = "tournament" | "stage" | "category" | "beatmap";
export type Severity = "info" | "warning" | "critical";

export interface Scope {
  type: ScopeType;
  id: string;
}

export interface Finding {
  severity: Severity;
  description: string;
  reason: string;
  recommendation: string;
  // ID of the Stage this finding is specifically about, when its own
  // scope (e.g. tournament) is broader than one stage. Absent when the
  // finding has no single target stage.
  targetStageId?: string;
}

export interface Citation {
  analyzerName: string;
  scope: Scope;
  finding: Finding;
}

export interface ReportSections {
  summary: string;
  findings: Citation[];
  warnings: Citation[];
  recommendations: string[];
  statistics: Record<string, number>;
}

export interface Report {
  scope: Scope;
  generatedAt: string;
  sections: ReportSections;
}

export interface Beatmap {
  id: string;
  title: string;
  artist: string;
  mapper: string;
  version: string;
  ar: number;
  od: number;
  cs: number;
  hp: number;
  bpm: number;
  coverUrl?: string;
}

// A slot's beatmap AR/OD/CS/HP/BPM/length as they actually play under the
// slot's own category's fixed mod (e.g. HR scales CS/AR/OD/HP; DT scales
// BPM/length and rescales AR/OD's timing windows) — computed server-side
// (backend/internal/modmap.EffectiveDifficultyFor) so the frontend never
// re-implements osu!'s mod-scaling formulas itself. Absent when the slot
// is unfilled or its category has no single fixed mod (FreeMod,
// Tiebreaker, unrecognized name) — there is no sound single value to show
// in either case.
export interface EffectiveDifficulty {
  ar: number;
  od: number;
  cs: number;
  hp: number;
  bpm: number;
  lengthSeconds: number;
  // Real, mod-specific Star Rating fetched from the osu! API at import
  // time — not something the frontend (or the AR/OD/CS/HP/BPM transform
  // above) can compute itself. Null when star rating fetching is disabled
  // or this mod combination hasn't been enriched yet.
  starRating: number | null;
}

export interface Slot {
  id: string;
  code: string; // e.g. "NM1", "HD2" — the slot's mod-category shorthand, as mappoolers write it
  beatmap: Beatmap | null;
  // Absent (not just null) on mock/sample data and on the couple of REST
  // client calls that don't have category context (see rest.ts's
  // assignBeatmap/clearBeatmap) — always present from a real getTournament.
  effectiveDifficulty?: EffectiveDifficulty | null;
}

export interface Category {
  id: string;
  name: string;
  order: number;
  slots: Slot[];
}

export interface Stage {
  id: string;
  name: string;
  order: number;
  categories: Category[];
  // The organizer's explicit target, or the stage's NM1 beatmap's star
  // rating if unset; null if neither is available.
  projectedStarRating: number | null;
}

export interface Tournament {
  id: string;
  name: string;
  edition: string;
  stages: Stage[];
}
