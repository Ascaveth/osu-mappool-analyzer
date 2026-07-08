// Central definition of the mod prefixes the UI recognizes out of the
// box. Tournament configuration is user-defined (CLAUDE.md) — a category
// with a mod prefix outside this list is still fully valid, it just falls
// back to an unstyled/unlabeled presentation instead of a guessed one.
// This list exists purely to make the common case (NM/HD/HR/DT/FM/TB)
// look nice; it must never be treated as the exhaustive set of allowed
// mod categories by validation logic.
export interface ModMeta {
  value: string;
  label: string;
  accent: string; // CSS var reference, e.g. "var(--mod-nm)"
}

export const KNOWN_MODS: ModMeta[] = [
  { value: "NM", label: "No Mod", accent: "var(--mod-nm)" },
  { value: "HD", label: "Hidden", accent: "var(--mod-hd)" },
  { value: "HR", label: "Hard Rock", accent: "var(--mod-hr)" },
  { value: "DT", label: "Double Time", accent: "var(--mod-dt)" },
  { value: "FM", label: "Free Mod", accent: "var(--mod-fm)" },
  { value: "TB", label: "Tiebreaker", accent: "var(--mod-tb)" },
];

const byValue: Record<string, ModMeta> = Object.fromEntries(
  KNOWN_MODS.map((m) => [m.value, m]),
);

// Returns the known label for a mod prefix, or the prefix itself for a
// custom/unrecognized one so the UI never shows a blank name.
export function modLabel(value: string): string {
  return byValue[value]?.label ?? value;
}

// Returns the accent color for a known mod prefix, or undefined for a
// custom one — callers should render unstyled rather than guess a color.
export function modAccent(value: string): string | undefined {
  return byValue[value]?.accent;
}
