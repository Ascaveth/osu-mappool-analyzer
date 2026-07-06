import type { Citation, Stage } from "@/lib/types";
import { MarginNote } from "./MarginNote";
import { formatBeatmapLabel, modAccentColor, slotAccentStyle } from "@/lib/beatmap-format";
import { sortBySeverity } from "@/lib/citation-labels";

/**
 * Renders a stage section with its categories and slots. Stage/tournament
 * findings are shown separately in FindingsSummary; category- and
 * beatmap-scoped findings render inline after each category's last slot.
 *
 * @param stage - The stage to display.
 * @param categoryNotes - Citations (category- and beatmap-scoped) grouped by category ID.
 * @param delay - Animation delay in milliseconds.
 */
export function StageSection({
  stage,
  categoryNotes,
  delay,
}: {
  stage: Stage;
  categoryNotes: Record<string, Citation[]>;
  delay: number;
}) {
  const slotCount = stage.categories.reduce((s, c) => s + c.slots.length, 0);

  return (
    <section
      id={`stage-${stage.id}`}
      className="stage reveal"
      style={{ animationDelay: `${delay}ms`, scrollMarginTop: "4.5rem" }}
    >
      <div className="stage-head">
        <div>
          <div className="stage-head-title">
            <h2 className="stage-name">{stage.name}</h2>
          </div>
          <p className="stage-meta">
            {stage.categories.length} categories · {slotCount} slots
          </p>
        </div>
      </div>

      {stage.categories.map((category) => {
        const notes = sortBySeverity(categoryNotes[category.id] ?? []);
        const slotCodeByBeatmapId = Object.fromEntries(
          category.slots.filter((s) => s.beatmap !== null).map((s) => [s.beatmap!.id, s.code]),
        );
        return (
          <div className="category-block" key={category.id}>
            <p className="category-name">
              {modAccentColor(category.slots[0]?.code ?? "") && (
                <span
                  className="category-dot"
                  style={{
                    background: modAccentColor(category.slots[0]?.code ?? ""),
                  }}
                />
              )}
              {category.name}
            </p>
            {category.slots.map((slot) => {
              const hasCover = !!slot.beatmap?.coverUrl;
              return (
                <div
                  className="slot-row"
                  key={slot.id}
                  style={slotAccentStyle(slot.code, slot.beatmap?.coverUrl)}
                >
                  <span className={`slot-code${hasCover ? " slot-chip" : ""}`}>
                    {slot.code}
                  </span>
                  <span className={`slot-title${hasCover ? " slot-chip" : ""}`}>
                    {slot.beatmap ? formatBeatmapLabel(slot.beatmap) : "— unfilled —"}
                  </span>
                  {slot.beatmap && (
                    <span className={`slot-stats${hasCover ? " slot-chip" : ""}`}>
                      AR {slot.beatmap.ar.toFixed(1)} · OD {slot.beatmap.od.toFixed(1)} ·{" "}
                      {slot.beatmap.bpm} BPM
                    </span>
                  )}
                </div>
              );
            })}
            {notes.length > 0 && (
              <div className="category-findings">
                {notes.map((c, i) => (
                  <MarginNote
                    key={i}
                    citation={c}
                    locationLabel={c.scope.type === "beatmap" ? slotCodeByBeatmapId[c.scope.id] : null}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })}
    </section>
  );
}
