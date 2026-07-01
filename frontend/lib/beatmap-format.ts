import type { CSSProperties } from "react";
import type { Beatmap } from "@/lib/types";

export function formatBeatmapLabel(
  bm: Pick<Beatmap, "artist" | "title" | "version">,
): string {
  return `${bm.artist} - ${bm.title} - [${bm.version}]`;
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
// mod's color, and (when a cover is available) a gradient from a
// mod-tinted paper tone on the left, where the code/title text sits, to
// the cover image on the right. The right edge never drops below ~45%
// tint — fully revealing the cover there would wash out the stats text.
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
    backgroundImage: `linear-gradient(to right, ${base} 0%, ${base} 30%, color-mix(in srgb, ${base} 45%, transparent) 100%), url(${coverUrl})`,
    backgroundSize: "cover",
    backgroundPosition: "center",
  };
}
