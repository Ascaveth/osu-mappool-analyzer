import type { CSSProperties } from "react";
import type { Beatmap } from "@/lib/types";

export function formatBeatmapLabel(
  bm: Pick<Beatmap, "artist" | "title" | "version">,
): string {
  return `${bm.artist} - ${bm.title} - [${bm.version}]`;
}

// Formats one AR/OD/CS/HP stat using the slot's mod-effective value (see
// lib/types.ts's EffectiveDifficulty) when available — that's what the
// slot is actually played at (e.g. HR-scaled AR/OD/CS/HP) — falling back
// to the beatmap's raw .osu value only when there's no effective value to
// show (unfilled slot, or a category with no single fixed mod).
export function formatStat(label: string, raw: number, effective: number | undefined): string {
  return `${label} ${(effective ?? raw).toFixed(1)}`;
}

// Same idea as formatStat, but for BPM: whole numbers.
export function formatBpm(raw: number, effective: number | undefined): string {
  return `${Math.round(effective ?? raw)} BPM`;
}

// Formats a slot's real, mod-specific Star Rating (see
// lib/types.ts's EffectiveDifficulty.starRating) — null when star rating
// fetching is disabled or this mod combination hasn't been enriched yet,
// shown as "SR --" rather than a misleading 0.
export function formatStarRating(starRating: number | null | undefined): string {
  return starRating == null ? "SR --" : `SR ${starRating.toFixed(2)}`;
}

const MOD_ACCENTS: Record<string, string> = {
  NM: "var(--mod-nm)",
  HD: "var(--mod-hd)",
  HR: "var(--mod-hr)",
  DT: "var(--mod-dt)",
  FM: "var(--mod-fm)",
  TB: "var(--mod-tb)",
};

// Slot codes are "{modPrefix}{index}", e.g. "HD2" — strip the trailing
// digits to recover the mod prefix a category/slot belongs to.
function modPrefix(code: string): string {
  return code.replace(/\d+$/, "");
}

// Returns the accent color for a slot/category's mod, or undefined for
// mods outside the known set (kept unstyled rather than guessing a color).
export function modAccentColor(code: string): string | undefined {
  return MOD_ACCENTS[modPrefix(code)];
}

// Renders a beatmap slot's row background: a left accent border in the
// mod's color, and (when a cover is available) the cover photo itself
// under a light ink veil — enough to keep every cover feeling like part
// of the same printed programme, not so much that the photo disappears.
// Text legibility over the photo is handled separately by `.slot-chip`
// (see globals.css), not by darkening the row.
export function slotAccentStyle(
  code: string,
  coverUrl: string | undefined,
): CSSProperties {
  const accent = modAccentColor(code);
  const base = accent
    ? `color-mix(in srgb, var(--paper) 88%, ${accent} 12%)`
    : "var(--paper)";

  const style: CSSProperties = accent ? { borderLeft: `3px solid ${accent}` } : {};

  if (!coverUrl) {
    if (accent) style.background = base;
    return style;
  }

  return {
    ...style,
    backgroundImage: `linear-gradient(to right, color-mix(in srgb, var(--ink) 18%, transparent) 0%, color-mix(in srgb, var(--ink) 8%, transparent) 100%), url(${coverUrl})`,
    backgroundSize: "cover",
    backgroundPosition: "center",
  };
}
