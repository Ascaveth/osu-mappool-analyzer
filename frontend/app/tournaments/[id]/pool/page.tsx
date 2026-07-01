"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Tournament } from "@/lib/types";
import { formatBeatmapLabel, modAccentColor, slotAccentStyle } from "@/lib/beatmap-format";

export default function PoolPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();

  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [slotInputs, setSlotInputs] = useState<Record<string, string>>({});
  const [slotImporting, setSlotImporting] = useState<Record<string, boolean>>({});
  const [slotErrors, setSlotErrors] = useState<Record<string, string>>({});

  const refresh = useCallback(async () => {
    try {
      const t = await api.getTournament(id);
      setTournament(t);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load pool");
    }
  }, [id]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const allSlots =
    tournament?.stages.flatMap((s) => s.categories.flatMap((c) => c.slots)) ??
    [];
  const filledCount = allSlots.filter((sl) => sl.beatmap !== null).length;
  const totalCount = allSlots.length;

  const importAndAssign = async (slotId: string) => {
    const url = (slotInputs[slotId] ?? "").trim();
    if (!url) return;
    setSlotImporting((prev) => ({ ...prev, [slotId]: true }));
    setSlotErrors((prev) => {
      const next = { ...prev };
      delete next[slotId];
      return next;
    });
    try {
      const bm = await api.importBeatmapFromUrl(url);
      await api.assignBeatmap(slotId, bm.id);
      setSlotInputs((prev) => {
        const next = { ...prev };
        delete next[slotId];
        return next;
      });
      await refresh();
    } catch (e) {
      setSlotErrors((prev) => ({
        ...prev,
        [slotId]: e instanceof Error ? e.message : "Import failed",
      }));
    } finally {
      setSlotImporting((prev) => ({ ...prev, [slotId]: false }));
    }
  };

  const clear = async (slotId: string) => {
    try {
      await api.clearBeatmap(slotId);
      setSlotInputs((prev) => {
        const next = { ...prev };
        delete next[slotId];
        return next;
      });
      setSlotErrors((prev) => {
        const next = { ...prev };
        delete next[slotId];
        return next;
      });
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to clear slot");
    }
  };

  const runAnalysis = async () => {
    setRunning(true);
    try {
      await api.getReport(id);
      router.push(`/tournaments/${id}/report`);
    } catch (e) {
      console.error(e);
      setRunning(false);
    }
  };

  if (error && !tournament) {
    return (
      <main className="programme">
        <p
          style={{
            color: "var(--mark)",
            fontFamily: "var(--font-data)",
            fontSize: "0.875rem",
          }}
        >
          Error: {error}
        </p>
        <Link
          href="/tournaments/new"
          style={{
            display: "inline-block",
            marginTop: "1rem",
            fontFamily: "var(--font-data)",
            fontSize: "0.6875rem",
            color: "var(--ink-soft)",
          }}
        >
          ← Back
        </Link>
      </main>
    );
  }

  if (!tournament) {
    return (
      <main className="programme">
        <p className="slot-stats">Loading pool…</p>
      </main>
    );
  }

  return (
    <main className="programme">
      <div className="masthead">
        <p className="masthead-eyebrow">
          <Link
            href="/tournaments/new"
            style={{ color: "inherit", textDecoration: "none" }}
          >
            ← Back
          </Link>
          {" · "}Step 2 of 2{" · "}
          {tournament.name}
          {tournament.edition ? ` ${tournament.edition}` : ""}
        </p>
        <h1 className="masthead-title">Pool Builder</h1>
      </div>

      {error && (
        <p
          style={{
            color: "var(--mark)",
            fontFamily: "var(--font-data)",
            fontSize: "0.8125rem",
            marginBottom: "1rem",
          }}
        >
          ▲ {error}
        </p>
      )}

      <div>
        {tournament.stages.map((stage) => (
          <section key={stage.id} style={{ marginBottom: "2.5rem" }}>
            <div
              style={{
                borderTop: "1px solid var(--ink)",
                paddingTop: "0.75rem",
                marginBottom: "0.5rem",
              }}
            >
              <h2 className="stage-name">{stage.name}</h2>
            </div>

            {stage.categories.map((cat) => (
              <div
                key={cat.id}
                style={{
                  borderTop: "1px solid var(--paper-line)",
                  paddingTop: "0.5rem",
                  paddingBottom: "0.5rem",
                }}
              >
                <p className="category-name">
                  {modAccentColor(cat.slots[0]?.code ?? "") && (
                    <span
                      className="category-dot"
                      style={{ background: modAccentColor(cat.slots[0]?.code ?? "") }}
                    />
                  )}
                  {cat.name}
                </p>
                {cat.slots.map((slot) => {
                  const hasCover = !!slot.beatmap?.coverUrl;
                  return (
                  <div key={slot.id}>
                    <div className="slot-line">
                      <div
                        className="slot-row slot-row--editable"
                        style={slotAccentStyle(slot.code, slot.beatmap?.coverUrl)}
                      >
                        <span className={`slot-code${hasCover ? " slot-chip" : ""}`}>
                          {slot.code}
                        </span>

                        {slot.beatmap ? (
                          <>
                            <span className={`slot-title${hasCover ? " slot-chip" : ""}`}>
                              {formatBeatmapLabel(slot.beatmap)}
                            </span>
                            <span className={`slot-stats${hasCover ? " slot-chip" : ""}`}>
                              AR {slot.beatmap.ar.toFixed(1)} · OD{" "}
                              {slot.beatmap.od.toFixed(1)} ·{" "}
                              {slot.beatmap.bpm} BPM
                            </span>
                            <button
                              className="btn btn-ghost pool-slot-clear"
                              onClick={() => clear(slot.id)}
                              title="Clear slot"
                              aria-label={`Clear beatmap from slot ${slot.code}`}
                            >
                              ×
                            </button>
                          </>
                        ) : (
                          <>
                            <input
                              className="field-input slot-input"
                              placeholder="paste beatmap URL or ID"
                              aria-label={`Beatmap URL or ID for slot ${slot.code}`}
                              value={slotInputs[slot.id] ?? ""}
                              onChange={(e) =>
                                setSlotInputs((prev) => ({
                                  ...prev,
                                  [slot.id]: e.target.value,
                                }))
                              }
                              onKeyDown={(e) => {
                                if (e.key === "Enter") importAndAssign(slot.id);
                              }}
                              disabled={!!slotImporting[slot.id]}
                            />
                            <button
                              className="btn btn-ghost pool-slot-confirm"
                              onClick={() => importAndAssign(slot.id)}
                              disabled={
                                !!slotImporting[slot.id] ||
                                !(slotInputs[slot.id] ?? "").trim()
                              }
                              title="Import & assign"
                              aria-label={`Import and assign beatmap to slot ${slot.code}`}
                            >
                              {slotImporting[slot.id] ? "…" : "✓"}
                            </button>
                          </>
                        )}
                      </div>
                    </div>
                    {slotErrors[slot.id] && (
                      <p className="slot-error">▲ {slotErrors[slot.id]}</p>
                    )}
                  </div>
                  );
                })}
              </div>
            ))}
          </section>
        ))}
      </div>

      <div className="wizard-nav">
        <span className="wizard-step-indicator">
          {filledCount} / {totalCount} slots filled
        </span>
        <button
          className="btn btn-primary"
          onClick={runAnalysis}
          disabled={running}
        >
          {running ? "Running Analysis…" : "Run Analysis →"}
        </button>
      </div>
    </main>
  );
}
