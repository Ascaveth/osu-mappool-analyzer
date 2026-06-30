import type { Citation } from "@/lib/types";

const GLYPH: Record<Citation["finding"]["severity"], string> = {
  critical: "●",
  warning: "▲",
  info: "·",
};

const ANALYZER_LABEL: Record<string, string> = {
  "composition-analyzer": "Composition",
  "progression-analyzer": "Progression",
  "balance-analyzer": "Balance",
  "diversity-analyzer": "Diversity",
};

export function MarginNote({ citation }: { citation: Citation }) {
  return (
    <div className="note">
      <span className="note-mark" aria-hidden="true">
        {GLYPH[citation.finding.severity]}
      </span>
      <div className="note-body">
        <p className="note-text">{citation.finding.description}.</p>
        <p className="note-why">{citation.finding.reason}.</p>
        <span className="note-source">
          {ANALYZER_LABEL[citation.analyzerName] ?? citation.analyzerName}
        </span>
      </div>
    </div>
  );
}
