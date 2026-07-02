import type { ReportSections } from "@/lib/types";
import { WipDisclaimer } from "@/components/WipDisclaimer";

/**
 * Renders the condensed analysis result count and the WIP disclaimer.
 *
 * @param sections - Report content and statistics to display
 */
export function ThesisHero({ sections }: { sections: ReportSections }) {
  const issueCount = sections.statistics.total_findings ?? 0;

  return (
    <section className="thesis reveal" style={{ animationDelay: "80ms" }}>
      <p className="results-line">
        Analysis results: {issueCount} issue{issueCount === 1 ? "" : "s"} found.
      </p>
      <WipDisclaimer />
    </section>
  );
}
