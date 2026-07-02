import type { Stage, Citation } from "@/lib/types";
import { MarginNote } from "./MarginNote";
import { formatBeatmapLabel, modAccentColor, slotAccentStyle } from "@/lib/beatmap-format";

/**
 * Renders a stage section with its categories, slots, and margin notes.
 *
 * @param stage - The stage to display.
 * @param stageNotes - Citations shown with the stage header.
 * @param categoryNotes - Citations grouped by category ID.
 * @param beatmapNotes - Citations grouped by beatmap ID.
 * @param delay - Animation delay in milliseconds.
 */
export function StageSection({
  stage,
  stageNotes,
  categoryNotes,
  beatmapNotes,
  delay,
}: {
  stage: Stage;
  stageNotes: Citation[];
  categoryNotes: Record<string, Citation[]>;
  beatmapNotes: Record<string, Citation[]>;
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
        {stageNotes.length > 0 && (
          <div className="marginalia">
            {stageNotes.map((c, i) => (
              <MarginNote key={i} citation={c} />
            ))}
          </div>
        )}
      </div>

      {stage.categories.map((category) => {
        const notes = categoryNotes[category.id] ?? [];
        return (
          <div
            className="category-block"
            key={category.id}
            style={notes.length === 0 ? { gridTemplateColumns: "1fr" } : undefined}
          >
            <div>
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
                const notes = slot.beatmap ? beatmapNotes[slot.beatmap.id] ?? [] : [];
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
                    {notes.length > 0 && (
                      <div className="marginalia">
                        {notes.map((c, i) => (
                          <MarginNote key={i} citation={c} />
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
            {notes.length > 0 && (
              <div className="marginalia">
                {notes.map((c, i) => (
                  <MarginNote key={i} citation={c} />
                ))}
              </div>
            )}
          </div>
        );
      })}
    </section>
  );
}
