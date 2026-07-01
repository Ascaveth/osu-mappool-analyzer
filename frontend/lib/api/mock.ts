import type { Tournament, Beatmap, Slot, Report, Stage, Category, Citation } from "@/lib/types";
import type { ApiClient } from "./client";
import type { CreateTournamentInput } from "./types";

const STORAGE_KEY = "osu-analyzer-mock";

const MOD_LABELS: Record<string, string> = {
  NM: "No Mod",
  HD: "Hidden",
  HR: "Hard Rock",
  DT: "Double Time",
  FM: "Free Mod",
  TB: "Tiebreaker",
};

interface Store {
  tournaments: Record<string, Tournament>;
  beatmaps: Record<string, Beatmap>;
  slotAssignments: Record<string, string>; // slotId → beatmapId
}

function emptyStore(): Store {
  return { tournaments: {}, beatmaps: {}, slotAssignments: {} };
}

function load(): Store {
  if (typeof window === "undefined") return emptyStore();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Store) : emptyStore();
  } catch {
    return emptyStore();
  }
}

function save(s: Store): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s));
  }
}

function uid(): string {
  return crypto.randomUUID();
}

function parseOsu(text: string): Omit<Beatmap, "id"> {
  const get = (k: string) =>
    text.match(new RegExp(`^${k}\\s*:\\s*(.+)$`, "m"))?.[1]?.trim() ?? "";
  const num = (k: string) => parseFloat(get(k)) || 0;

  let bpm = 0;
  const tpSection = text.match(/\[TimingPoints\]([\s\S]*?)(?:\n\[|$)/)?.[1] ?? "";
  for (const line of tpSection.trim().split("\n")) {
    const parts = line.split(",");
    // uninherited timing point: column 7 (index 6) === "1"
    if (parts.length >= 7 && parts[6]?.trim() === "1") {
      const beatLength = parseFloat(parts[1] ?? "0");
      if (beatLength > 0) {
        bpm = Math.round(60000 / beatLength);
        break;
      }
    }
  }

  // BeatmapSetID lives in the .osu file itself, so it's available
  // regardless of which URL form the user pasted (unlike a URL regex,
  // which only catches the beatmapsets/{setId}#{mode}/{diffId} form).
  const beatmapsetId = num("BeatmapSetID");
  const coverUrl =
    beatmapsetId > 0
      ? `https://assets.ppy.sh/beatmaps/${beatmapsetId}/covers/cover.jpg`
      : undefined;

  return {
    title: get("Title") || "Unknown Title",
    artist: get("Artist") || "Unknown Artist",
    mapper: get("Creator") || "Unknown Mapper",
    version: get("Version") || "Unknown Difficulty",
    ar: num("ApproachRate"),
    od: num("OverallDifficulty"),
    bpm,
    coverUrl,
  };
}

function extractBeatmapId(url: string): string {
  // https://osu.ppy.sh/beatmapsets/1555041#osu/3176982
  const setHash = url.match(/beatmapsets\/\d+#[a-z]+\/(\d+)/);
  if (setHash) return setHash[1];
  // https://osu.ppy.sh/beatmaps/3176982
  const beatmapPath = url.match(/beatmaps\/(\d+)/);
  if (beatmapPath) return beatmapPath[1];
  // bare ID
  const bare = url.trim().match(/^(\d+)$/);
  if (bare) return bare[1];
  throw new Error(`Cannot parse beatmap ID from URL: ${url}`);
}

function applyAssignments(t: Tournament, s: Store): Tournament {
  return {
    ...t,
    stages: t.stages.map((stage) => ({
      ...stage,
      categories: stage.categories.map((cat) => ({
        ...cat,
        slots: cat.slots.map((slot) => {
          const bmId = s.slotAssignments[slot.id];
          return { ...slot, beatmap: bmId ? (s.beatmaps[bmId] ?? null) : null };
        }),
      })),
    })),
  };
}

export function createMockClient(): ApiClient {
  return {
    async createTournament(input: CreateTournamentInput): Promise<Tournament> {
      const s = load();
      const t: Tournament = {
        id: uid(),
        name: input.name,
        edition: "",
        stages: input.stages.map((st): Stage => ({
          id: uid(),
          name: st.name,
          order: st.order,
          categories: st.categories.map((cat): Category => ({
            id: uid(),
            name: MOD_LABELS[cat.modPrefix] ?? cat.modPrefix,
            order: cat.order,
            slots: Array.from({ length: cat.slotCount }, (_, i) => ({
              id: uid(),
              code: `${cat.modPrefix}${i + 1}`,
              beatmap: null,
            })),
          })),
        })),
      };
      s.tournaments[t.id] = t;
      save(s);
      return t;
    },

    async getTournament(id: string): Promise<Tournament> {
      const s = load();
      const t = s.tournaments[id];
      if (!t) throw new Error(`Tournament "${id}" not found`);
      return applyAssignments(t, s);
    },

    async importBeatmapFromUrl(url: string): Promise<Beatmap> {
      const beatmapId = extractBeatmapId(url.trim());
      const res = await fetch(`/api/osu-proxy?id=${beatmapId}`);
      if (!res.ok) {
        const msg = await res.text().catch(() => res.statusText);
        throw new Error(`osu! fetch failed (${res.status}): ${msg}`);
      }
      const text = await res.text();
      const meta = parseOsu(text);
      const s = load();
      const key = `${meta.title}|${meta.version}|${meta.mapper}`;
      const existing = Object.values(s.beatmaps).find(
        (b) => `${b.title}|${b.version}|${b.mapper}` === key,
      );
      if (existing) return existing;
      const bm: Beatmap = { id: uid(), ...meta };
      s.beatmaps[bm.id] = bm;
      save(s);
      return bm;
    },

    async listBeatmaps(): Promise<Beatmap[]> {
      return Object.values(load().beatmaps);
    },

    async assignBeatmap(slotId: string, beatmapId: string): Promise<Slot> {
      const s = load();
      const beatmap = s.beatmaps[beatmapId];
      if (!beatmap) throw new Error(`Beatmap "${beatmapId}" not found`);
      for (const t of Object.values(s.tournaments)) {
        for (const stage of t.stages) {
          for (const cat of stage.categories) {
            const slot = cat.slots.find((sl) => sl.id === slotId);
            if (slot) {
              s.slotAssignments[slotId] = beatmapId;
              save(s);
              return { ...slot, beatmap };
            }
          }
        }
      }
      throw new Error(`Slot "${slotId}" not found`);
    },

    async clearBeatmap(slotId: string): Promise<Slot> {
      const s = load();
      delete s.slotAssignments[slotId];
      save(s);
      for (const t of Object.values(s.tournaments)) {
        for (const stage of t.stages) {
          for (const cat of stage.categories) {
            const slot = cat.slots.find((sl) => sl.id === slotId);
            if (slot) return { ...slot, beatmap: null };
          }
        }
      }
      throw new Error(`Slot "${slotId}" not found`);
    },

    async getReport(tournamentId: string): Promise<Report> {
      const s = load();
      const base = s.tournaments[tournamentId];
      if (!base) throw new Error(`Tournament "${tournamentId}" not found`);
      const t = applyAssignments(base, s);

      const allSlots = t.stages.flatMap((st) => st.categories.flatMap((c) => c.slots));
      const filled = allSlots.filter((sl) => sl.beatmap !== null);
      const unfilled = allSlots.filter((sl) => sl.beatmap === null);

      const findings: Citation[] = [];

      if (unfilled.length > 0) {
        findings.push({
          analyzerName: "validation-analyzer",
          scope: { type: "tournament", id: tournamentId },
          finding: {
            severity: "warning",
            description: `${unfilled.length} slot${unfilled.length !== 1 ? "s" : ""} left unfilled.`,
            reason: "Unfilled slots reduce analysis coverage.",
            recommendation: "Fill all slots before the final analysis run.",
          },
        });
      }

      const counts = filled.reduce<Record<string, number>>((acc, sl) => {
        const m = sl.beatmap!.mapper;
        acc[m] = (acc[m] ?? 0) + 1;
        return acc;
      }, {});
      for (const [mapper, n] of Object.entries(counts)) {
        if (n >= 3) {
          findings.push({
            analyzerName: "diversity-analyzer",
            scope: { type: "tournament", id: tournamentId },
            finding: {
              severity: "warning",
              description: `Mapper "${mapper}" appears ${n} times across the pool.`,
              reason: "High mapper repetition reduces pool diversity.",
              recommendation: `Replace some maps by ${mapper} with alternatives from other mappers.`,
            },
          });
        }
      }

      findings.push({
        analyzerName: "system",
        scope: { type: "tournament", id: tournamentId },
        finding: {
          severity: "info",
          description: "Running in demo mode — full analysis requires the backend server.",
          reason: "The Go analysis engine is not connected yet.",
          recommendation: "Start the backend server and replace the mock API client with the real one.",
        },
      });

      const issueCount = findings.filter(
        (f) => f.finding.severity === "warning" || f.finding.severity === "critical",
      ).length;

      return {
        scope: { type: "tournament", id: tournamentId },
        generatedAt: new Date().toISOString(),
        sections: {
          summary:
            filled.length > 0
              ? `${t.name} — ${filled.length} map${filled.length !== 1 ? "s" : ""} across ${t.stages.length} stage${t.stages.length !== 1 ? "s" : ""}. ${issueCount} issue${issueCount !== 1 ? "s" : ""} detected.`
              : "Pool is empty. Import beatmaps and assign them to slots to generate a real analysis.",
          findings,
          warnings: findings.filter(
            (f) => f.finding.severity === "warning" || f.finding.severity === "critical",
          ),
          recommendations: [...new Set(findings.map((f) => f.finding.recommendation))],
          statistics: {
            total_analyses: 2,
            total_findings: findings.length,
            findings_warning: findings.filter((f) => f.finding.severity === "warning").length,
            findings_critical: 0,
          },
        },
      };
    },
  };
}
