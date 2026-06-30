import type { Stage, Citation } from "@/lib/types";
import { MarginNote } from "./MarginNote";

const ROMAN = ["I", "II", "III", "IV", "V", "VI", "VII", "VIII"];

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
    <section className="stage reveal" style={{ animationDelay: `${delay}ms` }}>
      <div className="stage-head">
        <div>
          <div className="stage-head-title">
            <span className="stage-numeral">{ROMAN[stage.order - 1] ?? stage.order}</span>
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
          <div className="category-block" key={category.id}>
            <div>
              <p className="category-name">{category.name}</p>
              {category.slots.map((slot) => {
                const notes = slot.beatmap ? beatmapNotes[slot.beatmap.id] ?? [] : [];
                return (
                  <div className="slot-row" key={slot.id}>
                    <span className="slot-code">{slot.code}</span>
                    <span>
                      <span className="slot-title">{slot.beatmap?.title ?? "— unfilled —"}</span>
                      {slot.beatmap && (
                        <>
                          {" "}
                          <span className="slot-artist">— {slot.beatmap.artist}</span>
                        </>
                      )}
                    </span>
                    {slot.beatmap && (
                      <span className="slot-stats">
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
