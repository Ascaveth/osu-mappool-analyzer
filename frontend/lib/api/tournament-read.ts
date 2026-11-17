import type { Beatmap, Tournament, Stage, Category, Slot, EffectiveDifficulty } from "@/lib/types";

interface WireEffectiveDifficulty {
  ar: number;
  od: number;
  cs: number;
  hp: number;
  bpm: number;
  length_seconds: number;
  star_rating: number | null;
}

export interface WireSlot {
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

export interface WireTournament {
  id: string;
  name: string;
  edition: string;
  stages: WireStage[];
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

export function toTournament(w: WireTournament, beatmapsById: Map<string, Beatmap>): Tournament {
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


export async function readTournament(wire: WireTournament, loadBeatmaps: (ids: string[]) => Promise<Map<string, Beatmap>>): Promise<Tournament> {
  return toTournament(wire, await loadBeatmaps(collectBeatmapIds(wire)));
}
