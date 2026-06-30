import { Masthead } from "@/components/Masthead";
import { ThesisHero } from "@/components/ThesisHero";
import { StageSection } from "@/components/StageSection";
import { tournament, report } from "@/lib/sample-data";
import type { Citation } from "@/lib/types";

export default function Home() {
  const { findings } = report.sections;

  // A tournament-scope (e.g. progression) finding is shown under the
  // stage its description names last — for a regression/spike
  // description ("...drops from X ("A") to Y ("B")"), that's the stage
  // the change lands on, not the one it started from.
  const targetStageName = (description: string) => {
    const matches = [...description.matchAll(/“([^”]+)”/g)];
    return matches.at(-1)?.[1];
  };

  const stageNotes = (stageId: string, stageName: string): Citation[] =>
    findings.filter(
      (c) =>
        (c.scope.type === "stage" && c.scope.id === stageId) ||
        (c.scope.type === "tournament" && targetStageName(c.finding.description) === stageName),
    );

  const categoryNotesFor = (stage: (typeof tournament.stages)[number]) =>
    Object.fromEntries(
      stage.categories.map((category) => [
        category.id,
        findings.filter((c) => c.scope.type === "category" && c.scope.id === category.id),
      ]),
    );

  return (
    <main className="programme">
      <Masthead tournament={tournament} report={report} />
      <ThesisHero sections={report.sections} />
      {tournament.stages.map((stage, i) => (
        <StageSection
          key={stage.id}
          stage={stage}
          stageNotes={stageNotes(stage.id, stage.name)}
          categoryNotes={categoryNotesFor(stage)}
          delay={160 + i * 90}
        />
      ))}
      <p className="colophon">⁂ End of report · {tournament.name} {tournament.edition}</p>
    </main>
  );
}
