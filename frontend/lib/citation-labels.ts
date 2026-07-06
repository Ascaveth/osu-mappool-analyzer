import type { Citation, Severity, Tournament } from "@/lib/types";

export const SEVERITY_ORDER: Severity[] = ["critical", "warning", "info"];

export function sortBySeverity(citations: Citation[]): Citation[] {
  return [...citations].sort(
    (a, b) =>
      SEVERITY_ORDER.indexOf(a.finding.severity) - SEVERITY_ORDER.indexOf(b.finding.severity),
  );
}

// Resolves a Citation's scope into a short, human-readable location tag
// (e.g. "Round of 16 · NM3") so a finding stays traceable once it's no
// longer rendered next to the thing it's about.
export function citationLocation(citation: Citation, tournament: Tournament): string | null {
  const { scope, finding } = citation;

  if (scope.type === "tournament") {
    return finding.targetStageId
      ? (tournament.stages.find((s) => s.id === finding.targetStageId)?.name ?? null)
      : null;
  }

  if (scope.type === "stage") {
    return tournament.stages.find((s) => s.id === scope.id)?.name ?? null;
  }

  if (scope.type === "category") {
    for (const stage of tournament.stages) {
      const category = stage.categories.find((c) => c.id === scope.id);
      if (category) return `${stage.name} · ${category.name}`;
    }
    return null;
  }

  // scope.type === "beatmap"
  for (const stage of tournament.stages) {
    for (const category of stage.categories) {
      const slot = category.slots.find((s) => s.beatmap?.id === scope.id);
      if (slot) return `${stage.name} · ${slot.code}`;
    }
  }
  return null;
}
