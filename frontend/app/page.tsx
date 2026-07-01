import Link from "next/link";

export default function Home() {
  return (
    <main className="programme">
      <div className="masthead">
        <p className="masthead-eyebrow">osu! Mappool Analyzer</p>
        <h1 className="masthead-title">
          Analysis Engine
        </h1>
      </div>

      <p
        style={{
          fontFamily: "var(--font-display)",
          fontStyle: "italic",
          fontSize: "clamp(1.25rem, 2.4vw, 1.75rem)",
          lineHeight: 1.45,
          maxWidth: "42rem",
          marginBottom: "2.5rem",
          color: "var(--ink-soft)",
        }}
      >
        Evaluate tournament mappools and generate structured insights for
        organizers and mappoolers.
      </p>

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
