import type { Tournament, Report } from "@/lib/types";

/**
 * Formats an ISO date string as a US-English long date.
 *
 * @param iso - The ISO date string to format
 * @returns The date formatted with a long month name, numeric day, and numeric year
 */
function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

/**
 * Renders the tournament report masthead.
 *
 * @param tournament - Tournament data used for the title and slot counts.
 * @param report - Report data used for the generated date.
 * @returns The masthead header element.
 */
export function Masthead({ tournament, report }: { tournament: Tournament; report: Report }) {
  const slotCount = tournament.stages.reduce(
    (total, stage) => total + stage.categories.reduce((s, c) => s + c.slots.length, 0),
    0,
  );

  return (
    <header className="masthead reveal">
      <p className="masthead-eyebrow">Mappool Analysis · Tournament Report</p>
      <h1 className="masthead-title">
        {tournament.name} <em>{tournament.edition}</em>
      </h1>
      <p className="stage-meta" style={{ marginTop: "0.6rem" }}>
        Generated {formatDate(report.generatedAt)} · {tournament.stages.length} stages ·{" "}
        {slotCount} slots
      </p>
    </header>
  );
}
