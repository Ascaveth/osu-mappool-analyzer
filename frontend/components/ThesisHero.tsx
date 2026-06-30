import type { ReportSections } from "@/lib/types";

const STAT_LABELS: Record<string, string> = {
  total_analyses: "Analyses run",
  total_findings: "Findings",
  findings_warning: "Warnings",
  findings_critical: "Critical",
};

/**
 * Renders the thesis summary and statistic totals.
 *
 * @param sections - Report content and statistics to display
 */
export function ThesisHero({ sections }: { sections: ReportSections }) {
  return (
    <section className="thesis reveal" style={{ animationDelay: "80ms" }}>
      <p className="thesis-text">{sections.summary}</p>
      <div className="stat-ledger">
        {Object.entries(STAT_LABELS).map(([key, label]) => (
          <span key={key}>
            <strong>{sections.statistics[key] ?? 0}</strong> {label}
          </span>
        ))}
      </div>
    </section>
  );
}
