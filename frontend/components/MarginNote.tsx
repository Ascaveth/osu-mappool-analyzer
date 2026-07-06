import type { Citation } from "@/lib/types";

const GLYPH: Record<Citation["finding"]["severity"], string> = {
  critical: "●",
  warning: "▲",
  info: "·",
};

// Analyzers write descriptions lowercase (mid-sentence style, e.g. for
// composing into summary text); capitalize the first letter here so it
// reads correctly as its own sentence in the report.
function capitalize(text: string): string {
  return text.length > 0 ? text[0].toUpperCase() + text.slice(1) : text;
}

/**
 * Renders a margin note for a citation.
 *
 * @param citation - The citation data to display
 * @param locationLabel - Optional resolved scope location (e.g. "Round of 16 · NM3")
 * @returns A margin note showing the severity marker, finding text, and location
 */
export function MarginNote({
  citation,
  locationLabel,
}: {
  citation: Citation;
  locationLabel?: string | null;
}) {
  return (
    <div className="note">
      <span
        className={`note-mark note-mark--${citation.finding.severity}`}
        aria-hidden="true"
      >
        {GLYPH[citation.finding.severity]}
      </span>
      <div className="note-body">
        <p className="note-text">{capitalize(citation.finding.description)}.</p>
        {locationLabel && <span className="note-source">{locationLabel}</span>}
      </div>
    </div>
  );
}
