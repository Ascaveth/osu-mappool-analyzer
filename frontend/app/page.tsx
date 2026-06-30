import { Masthead } from "@/components/Masthead";
import { ThesisHero } from "@/components/ThesisHero";
import { StageSection } from "@/components/StageSection";
import { MarginNote } from "@/components/MarginNote";
import { tournament, report } from "@/lib/sample-data";
import type { Citation } from "@/lib/types";

/**
 * Renders the home page for the tournament report.
 */
export default function Home() {
  const { findings } = report.sections;

  // A tournament-scope (e.g. progression) finding is shown under the
  // stage it's specifically about, per the analyzer-supplied
  // targetStageId — e.g. for a regression/spike finding, that's the
  // stage the change lands on, not the one it started from.
  const stageNotes = (stageId: string): Citation[] =>
    findings.filter(
      (c) =>
        (c.scope.type === "stage" && c.scope.id === stageId) ||
        (c.scope.type === "tournament" && c.finding.targetStageId === stageId),
    );

  const categoryNotesFor = (stage: (typeof tournament.stages)[number]) =>
    Object.fromEntries(
      stage.categories.map((category) => [
        category.id,
        findings.filter((c) => c.scope.type === "category" && c.scope.id === category.id),
      ]),
    );

  const beatmapNotesFor = (stage: (typeof tournament.stages)[number]) =>
    Object.fromEntries(
      stage.categories
        .flatMap((category) => category.slots)
        .filter((s) => s.beatmap !== null)
        .map((s) => [
          s.beatmap!.id,
          findings.filter((c) => c.scope.type === "beatmap" && c.scope.id === s.beatmap!.id),
        ]),
    );

  // Tournament-scope findings without a single stage to point to (no
  // targetStageId) would otherwise never render anywhere — surface them
  // as a standalone fallback rather than silently dropping them.
  const tournamentWideNotes = findings.filter(
    (c) => c.scope.type === "tournament" && !c.finding.targetStageId,
  );

  return (
    <main className="programme">
      <Masthead tournament={tournament} report={report} />
      <ThesisHero sections={report.sections} />
      {tournamentWideNotes.length > 0 && (
        <div className="marginalia">
          {tournamentWideNotes.map((c, i) => (
            <MarginNote key={i} citation={c} />
          ))}
        </div>
      )}
      {tournament.stages.map((stage, i) => (
        <StageSection
          key={stage.id}
          stage={stage}
          stageNotes={stageNotes(stage.id)}
          categoryNotes={categoryNotesFor(stage)}
          beatmapNotes={beatmapNotesFor(stage)}
          delay={160 + i * 90}
        />
      ))}
      <p className="colophon">⁂ End of report · {tournament.name} {tournament.edition}</p>
    </main>
  );
}
