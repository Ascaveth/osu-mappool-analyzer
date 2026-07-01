import Link from "next/link";
import { MarginNote } from "@/components/MarginNote";
import { report as sampleReport } from "@/lib/sample-data";

const ro16Findings = sampleReport.sections.findings.filter(
  (c) =>
    (c.scope.type === "stage" && c.scope.id === "stage-ro16") ||
    (c.scope.type === "tournament" && c.finding.targetStageId === "stage-ro16"),
);

export default function Home() {
  return (
    <main className="programme">
      <div className="masthead">
        <h1 className="masthead-title">osu! Mappool Analyzer</h1>
        <h2 className="masthead-eyebrow">Ngakak abis Boss</h2>
      </div>

      <p
        style={{
          fontFamily: "var(--font-display)",
          fontStyle: "italic",
          fontSize: "clamp(1.25rem, 2.4vw, 1.75rem)",
          lineHeight: 1.45,
          maxWidth: "42rem",
          marginBottom: "2.25rem",
          color: "var(--ink-soft)",
        }}
      >
        Why need to testplay if you can use an automated mappool analyzer?
      </p>

      <section className="exhibit reveal" style={{ animationDelay: "100ms" }}>
        <div className="exhibit-head">
          <h2 className="stage-name">Round of 16</h2>
        </div>
        <div className="exhibit-notes">
          {ro16Findings.map((c) => (
            <MarginNote key={`${c.analyzerName}-${c.scope.type}-${c.scope.id}`} citation={c} />
          ))}
        </div>
        <p className="exhibit-caption">
          {ro16Findings.length} finding{ro16Findings.length === 1 ? "" : "s"} from a generated analysis, Ascaveth Invitational Tournament 2023
        </p>
      </section>

      <Link href="/tournaments/new" className="btn btn-primary">
        Analyze a Mappool →
      </Link>

      <p
        className="colophon"
        style={{ marginTop: "4rem", borderTop: "1px solid var(--paper-line)", paddingTop: "1.5rem" }}
      >
        Demo mode · Full analysis requires the backend server
      </p>
    </main>
  );
}
