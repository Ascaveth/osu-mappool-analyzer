import type { Citation, Tournament, Severity } from "@/lib/types";
import { MarginNote } from "./MarginNote";
import { citationLocation, SEVERITY_ORDER } from "@/lib/citation-labels";

const SEVERITY_LABEL: Record<Severity, string> = {
  critical: "Critical",
  warning: "Warning",
  info: "Info",
};

/**
 * Renders the report's headline findings (tournament- and stage-scoped) as
 * one bullet list grouped by severity, right below the results line.
 * Category- and beatmap-scoped findings are rendered separately, inline at
 * the bottom of their category block (see StageSection).
 *
 * @param citations - Tournament- and stage-scoped findings to summarize
 * @param tournament - Used to resolve each finding's scope into a location label
 */
export function FindingsSummary({
  citations,
  tournament,
}: {
  citations: Citation[];
  tournament: Tournament;
}) {
  if (citations.length === 0) return null;

  const groups = SEVERITY_ORDER.map((severity) => ({
    severity,
    items: citations.filter((c) => c.finding.severity === severity),
  })).filter((g) => g.items.length > 0);

  return (
    <section className="findings-summary reveal" style={{ animationDelay: "120ms" }}>
      {groups.map(({ severity, items }) => (
        <div key={severity} className="findings-summary-group">
          <p className="findings-summary-heading">
            {SEVERITY_LABEL[severity]} ({items.length})
          </p>
          <ul className="findings-summary-list">
            {items.map((c, i) => (
              <li key={i}>
                <MarginNote citation={c} locationLabel={citationLocation(c, tournament)} />
              </li>
            ))}
          </ul>
        </div>
      ))}
    </section>
  );
}
