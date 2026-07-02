/**
 * Renders the analyzer's work-in-progress disclaimer.
 *
 * Shown wherever analysis findings are presented, so nobody mistakes a
 * generated finding for a verdict.
 */
export function WipDisclaimer() {
  return (
    <p className="disclaimer">
      <span className="disclaimer-eyebrow">Notes</span>
        This analyzer is a work in progress. 
        Always review each suggestion and decide whether it fits your mappools.
    </p>
  );
}
