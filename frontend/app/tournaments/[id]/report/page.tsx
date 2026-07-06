"use client";

import { use, useState, useEffect } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { Masthead } from "@/components/Masthead";
import { ThesisHero } from "@/components/ThesisHero";
import { StageSection } from "@/components/StageSection";
import { StageNav } from "@/components/StageNav";
import { FindingsSummary } from "@/components/FindingsSummary";
import type { Tournament, Report, Citation } from "@/lib/types";

// Analyzers can independently flag the same underlying issue for the same
// scope; collapse those into a single citation so the report doesn't show
// the same finding twice.
function dedupeCitations(citations: Citation[]): Citation[] {
  const seen = new Set<string>();
  return citations.filter((c) => {
    const key = `${c.analyzerName}|${c.scope.type}|${c.scope.id}|${c.finding.description}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export default function ReportPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [report, setReport] = useState<Report | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const [t, r] = await Promise.all([api.getTournament(id), api.getReport(id)]);
      return { t, r };
    })()
      .then(({ t, r }) => {
        if (cancelled) return;
        setTournament(t);
        setReport(r);
        setError(null);
      })
      .catch((e) => {
        if (cancelled) return;
        setTournament(null);
        setReport(null);
        setError(e instanceof Error ? e.message : "Failed to load report");
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (error) {
    return (
      <main className="programme">
        <div className="alert" role="alert">
          <span className="alert-icon" aria-hidden="true">▲</span>
          <p className="alert-text">Error: {error}</p>
        </div>
        <Link
          href="/"
          style={{
            display: "inline-block",
            marginTop: "1rem",
            fontFamily: "var(--font-data)",
            fontSize: "0.6875rem",
            color: "var(--ink-soft)",
          }}
        >
          ← Home
        </Link>
      </main>
    );
  }

  if (!tournament || !report) {
    return (
      <main className="programme">
        <p className="slot-stats">Generating report…</p>
      </main>
    );
  }

  const findings = dedupeCitations(report.sections.findings);

  // Headline findings: shown as one severity-grouped block right below the
  // results line. Category/beatmap-scoped findings render separately,
  // inline at the bottom of their category block (see StageSection).
  const headlineFindings = findings.filter(
    (c) => c.scope.type === "tournament" || c.scope.type === "stage",
  );

  const categoryNotesFor = (stage: Tournament["stages"][number]) =>
    Object.fromEntries(
      stage.categories.map((cat) => {
        const beatmapIds = new Set(
          cat.slots.filter((s) => s.beatmap !== null).map((s) => s.beatmap!.id),
        );
        return [
          cat.id,
          findings.filter(
            (c) =>
              (c.scope.type === "category" && c.scope.id === cat.id) ||
              (c.scope.type === "beatmap" && beatmapIds.has(c.scope.id)),
          ),
        ];
      }),
    );

  return (
    <main className="programme">
      <div style={{ marginBottom: "1.5rem" }}>
        <Link
          href={`/tournaments/${id}/pool`}
          style={{
            fontFamily: "var(--font-data)",
            fontSize: "0.6875rem",
            letterSpacing: "0.08em",
            textTransform: "uppercase",
            color: "var(--ink-soft)",
            textDecoration: "none",
          }}
        >
          ← Import Beatmaps
        </Link>
      </div>

      <Masthead tournament={tournament} report={report} />
      <StageNav stages={tournament.stages} />
      <ThesisHero sections={report.sections} />
      <FindingsSummary citations={headlineFindings} tournament={tournament} />

      {tournament.stages.map((stage, i) => (
        <StageSection
          key={stage.id}
          stage={stage}
          categoryNotes={categoryNotesFor(stage)}
          delay={160 + i * 90}
        />
      ))}

      <p className="footer-note">
        ⁂ End of report · {tournament.name}
        {tournament.edition ? ` ${tournament.edition}` : ""}
      </p>
    </main>
  );
}
