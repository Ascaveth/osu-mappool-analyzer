import Link from "next/link";
import { HowToUse } from "@/components/HowToUse";
import { WipDisclaimer } from "@/components/WipDisclaimer";

export default function Home() {
  return (
    <main className="programme">
      <div className="masthead">
        <h1 className="masthead-title">osu! Mappool Analyzer</h1>
        <h2 className="masthead-eyebrow">Tournament mappool analysis</h2>
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
        Ready to flame the #mappool-feedback channel?
      </p>

      <HowToUse />

      <Link href="/tournaments/new" className="btn btn-primary">
        Analyze a Mappool →
      </Link>

      <WipDisclaimer />

      <p
        className="footer-note"
        style={{ marginTop: "4rem", borderTop: "1px solid var(--paper-line)", paddingTop: "1.5rem" }}
      >
        Alpha state
        <span className="footer-note-detail">
          This is an early build — features may change or break.
        </span>
      </p>
    </main>
  );
}
