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
  bpm: number;
  coverUrl?: string;
}

export interface Slot {
  id: string;
  code: string; // e.g. "NM1", "HD2" — the slot's mod-category shorthand, as mappoolers write it
  beatmap: Beatmap | null;
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
}

export interface Tournament {
  id: string;
  name: string;
  edition: string;
  stages: Stage[];
}
