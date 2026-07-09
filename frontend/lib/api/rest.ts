// Real ApiClient backed by the Go server's REST API (docs/api/openapi.yaml).
// Each method fetches the corresponding endpoint and translates the wire
// (snake_case) shape into lib/types.ts's camelCase presentation types — the
// same translation-at-the-boundary role internal/normalize plays for .osu
// syntax on the backend, just done here in TypeScript.
import type {
  Tournament,
  Stage,
  Category,
  Slot,
  Beatmap,
  EffectiveDifficulty,
  Report,
  Citation,
  Finding,
} from "@/lib/types";
import { extractBeatmapId } from "@/lib/beatmap-id";
import type { ApiClient } from "./client";
import { ApiError } from "./client";
import type { CreateTournamentInput } from "./types";

interface WireProblem {
  type?: string;
  title: string;
  status: number;
  detail?: string;
}

interface WireEffectiveDifficulty {
  ar: number;
  od: number;
  cs: number;
  hp: number;
  bpm: number;
  length_seconds: number;
  star_rating: number | null;
}

interface WireSlot {
  id: string;
  position: number;
  beatmap_id: string | null;
  effective_difficulty: WireEffectiveDifficulty | null;
}

interface WireCategory {
  id: string;
  name: string;
  order: number;
  slots: WireSlot[];
}

interface WireStage {
  id: string;
  name: string;
  order: number;
  categories: WireCategory[];
  projected_star_rating: number | null;
}

interface WireTournament {
  id: string;
  name: string;
  edition: string;
  stages: WireStage[];
}

interface WireBeatmap {
  id: string;
  title: string;
  artist: string;
  mapper: string;
  version: string;
  tags: string[];
  ar: number;
  od: number;
  cs: number;
  hp: number;
  bpm: number;
  star_rating: number;
  length_seconds: number;
  object_count: number;
  slider_ratio: number;
  osu_file_hash: string;
}

interface WireFinding {
  severity: Finding["severity"];
  description: string;
  reason: string;
  recommendation: string;
  metrics?: Record<string, number>;
}

interface WireCitation {
  analyzer_name: string;
  scope: { type: Citation["scope"]["type"]; id: string };
  finding: WireFinding;
}

interface WireReport {
  scope: { type: Citation["scope"]["type"]; id: string };
  generated_at: string;
  sections: {
    summary: string;
    findings: WireCitation[];
    warnings: WireCitation[];
    recommendations: string[];
    statistics: Record<string, number>;
  };
}

interface WireListResponse<T> {
  data: T[];
  pagination: { next_cursor: string | null; has_more: boolean };
}

async function request(baseUrl: string, path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(`${baseUrl}${path}`, init);
  if (!res.ok) {
    let detail = res.statusText;
    let type: string | undefined;
    const contentType = res.headers.get("Content-Type") ?? "";
    if (contentType.includes("json")) {
      try {
        const problem = (await res.json()) as WireProblem;
        detail = problem.detail || problem.title || detail;
        type = problem.type;
      } catch {
        // fall through to statusText
      }
    }
    throw new ApiError(res.status, detail, type);
  }
  return res;
}

function toBeatmap(w: WireBeatmap, coverUrl?: string): Beatmap {
  return {
    id: w.id,
    title: w.title,
    artist: w.artist,
    mapper: w.mapper,
    version: w.version,
    ar: w.ar,
    od: w.od,
    cs: w.cs,
    hp: w.hp,
    bpm: w.bpm,
    coverUrl,
  };
}

function toEffectiveDifficulty(w: WireEffectiveDifficulty | null): EffectiveDifficulty | null {
  if (!w) return null;
  return {
    ar: w.ar,
    od: w.od,
    cs: w.cs,
    hp: w.hp,
    bpm: w.bpm,
    lengthSeconds: w.length_seconds,
    starRating: w.star_rating,
  };
}

function toSlot(w: WireSlot, categoryName: string, beatmapsById: Map<string, Beatmap>): Slot {
  return {
    id: w.id,
    code: `${categoryName}${w.position}`,
    beatmap: w.beatmap_id ? (beatmapsById.get(w.beatmap_id) ?? null) : null,
    effectiveDifficulty: toEffectiveDifficulty(w.effective_difficulty),
  };
}

function toCategory(w: WireCategory, beatmapsById: Map<string, Beatmap>): Category {
  return {
    id: w.id,
    name: w.name,
    order: w.order,
    slots: w.slots.map((s) => toSlot(s, w.name, beatmapsById)),
  };
}

function toStage(w: WireStage, beatmapsById: Map<string, Beatmap>): Stage {
  return {
    id: w.id,
    name: w.name,
    order: w.order,
    categories: w.categories.map((c) => toCategory(c, beatmapsById)),
    projectedStarRating: w.projected_star_rating ?? null,
  };
}

function toTournament(w: WireTournament, beatmapsById: Map<string, Beatmap>): Tournament {
  return {
    id: w.id,
    name: w.name,
    edition: w.edition,
    stages: w.stages.map((s) => toStage(s, beatmapsById)),
  };
}

function collectBeatmapIds(w: WireTournament): string[] {
  const ids = new Set<string>();
  for (const stage of w.stages) {
    for (const cat of stage.categories) {
      for (const slot of cat.slots) {
        if (slot.beatmap_id) ids.add(slot.beatmap_id);
      }
    }
  }
  return [...ids];
}

function toFinding(w: WireFinding): Finding {
  return {
    severity: w.severity,
    description: w.description,
    reason: w.reason,
    recommendation: w.recommendation,
  };
}

function toCitation(w: WireCitation): Citation {
  return {
    analyzerName: w.analyzer_name,
    scope: w.scope,
    finding: toFinding(w.finding),
  };
}

function toReport(w: WireReport): Report {
  return {
    scope: w.scope,
    generatedAt: w.generated_at,
    sections: {
      summary: w.sections.summary,
      findings: w.sections.findings.map(toCitation),
      warnings: w.sections.warnings.map(toCitation),
      recommendations: w.sections.recommendations,
      statistics: w.sections.statistics,
    },
  };
}

// coverUrl is a pure presentation value never sourced from the backend
// (lib/types.ts's own header comment) — extracted client-side from the
// beatmapset ID embedded in the raw .osu file, the same way mock.ts does.
function extractCoverUrl(osuFileText: string): string | undefined {
  const setId = osuFileText.match(/^BeatmapSetID\s*:\s*(\d+)/m)?.[1];
  if (!setId || setId === "0" || setId === "-1") return undefined;
  return `https://assets.ppy.sh/beatmaps/${setId}/covers/cover.jpg`;
}

import { extractBeatmapId } from "@/lib/beatmap-id";

export function createRestClient(baseUrl: string): ApiClient {
  const beatmapCovers = new Map<string, string>();

  async function fetchBeatmapsById(ids: string[]): Promise<Map<string, Beatmap>> {
    const map = new Map<string, Beatmap>();
    await Promise.all(
      ids.map(async (id) => {
        const res = await request(baseUrl, `/beatmaps/${id}`);
        const wire = (await res.json()) as WireBeatmap;
        map.set(id, toBeatmap(wire, beatmapCovers.get(id)));
      }),
    );
    return map;
  }

  return {
    async createTournament(input: CreateTournamentInput): Promise<Tournament> {
      const body = {
        name: input.name,
        edition: "",
        stages: input.stages.map((st) => ({
          name: st.name,
          order: st.order,
          categories: st.categories.map((cat) => ({
            name: cat.modPrefix,
            order: cat.order,
            slotCount: cat.slotCount,
          })),
          projectedStarRating: st.projectedStarRating,
        })),
      };
      const res = await request(baseUrl, "/tournaments", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const wire = (await res.json()) as WireTournament;
      return toTournament(wire, new Map());
    },

    async getTournament(id: string): Promise<Tournament> {
      const res = await request(baseUrl, `/tournaments/${id}`);
      const wire = (await res.json()) as WireTournament;
      const beatmapsById = await fetchBeatmapsById(collectBeatmapIds(wire));
      return toTournament(wire, beatmapsById);
    },

    async importBeatmapFromUrl(url: string): Promise<Beatmap> {
      const beatmapId = extractBeatmapId(url.trim());
      const osuRes = await fetch(`/api/osu-proxy?id=${beatmapId}`);
      if (!osuRes.ok) {
        const msg = await osuRes.text().catch(() => osuRes.statusText);
        throw new Error(`osu! fetch failed (${osuRes.status}): ${msg}`);
      }
      const text = await osuRes.text();
      const coverUrl = extractCoverUrl(text);

      const form = new FormData();
      form.append("file", new Blob([text], { type: "text/plain" }), `${beatmapId}.osu`);

      const res = await request(baseUrl, "/beatmaps", { method: "POST", body: form });
      const wire = (await res.json()) as WireBeatmap;
      if (coverUrl) beatmapCovers.set(wire.id, coverUrl);
      return toBeatmap(wire, coverUrl);
    },

    async listBeatmaps(): Promise<Beatmap[]> {
      const all: Beatmap[] = [];
      let cursor: string | null = null;
      do {
        const qs = cursor ? `?cursor=${encodeURIComponent(cursor)}&limit=100` : "?limit=100";
        const res = await request(baseUrl, `/beatmaps${qs}`);
        const page = (await res.json()) as WireListResponse<WireBeatmap>;
        all.push(...page.data.map((b) => toBeatmap(b, beatmapCovers.get(b.id))));
        cursor = page.pagination.has_more ? page.pagination.next_cursor : null;
      } while (cursor);
      return all;
    },

    async assignBeatmap(slotId: string, beatmapId: string): Promise<Slot> {
      const res = await request(baseUrl, `/slots/${slotId}/beatmap`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ beatmap_id: beatmapId }),
      });
      const wire = (await res.json()) as WireSlot;
      const beatmapRes = await request(baseUrl, `/beatmaps/${beatmapId}`);
      const beatmapWire = (await beatmapRes.json()) as WireBeatmap;
      // The wire Slot doesn't carry its owning category's name, so `code`
      // can't be derived here — callers (see pool/page.tsx) always re-fetch
      // the whole Tournament via getTournament after assigning, which does
      // have that context, and don't use this return value's `code`.
      return {
        id: wire.id,
        code: "",
        beatmap: toBeatmap(beatmapWire, beatmapCovers.get(beatmapId)),
        effectiveDifficulty: null,
      };
    },

    async clearBeatmap(slotId: string): Promise<Slot> {
      await request(baseUrl, `/slots/${slotId}/beatmap`, { method: "DELETE" });
      return { id: slotId, code: "", beatmap: null, effectiveDifficulty: null };
    },

    async getReport(tournamentId: string): Promise<Report> {
      const res = await request(baseUrl, `/tournaments/${tournamentId}/report`);
      const wire = (await res.json()) as WireReport;
      return toReport(wire);
    },
  };
}
