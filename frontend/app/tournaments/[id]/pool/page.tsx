"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Tournament } from "@/lib/types";
import {
  formatBeatmapLabel,
  formatBpm,
  formatStarRating,
  formatStat,
  modAccentColor,
  slotAccentStyle,
} from "@/lib/beatmap-format";
import {
  previewBulkPaste,
  stageTotalSlots,
  type BulkPasteAssignment,
  type BulkPastePreview,
} from "@/lib/bulk-paste";
import { ClipboardCheck, Trash2 } from "lucide-react";

type StageBulkState = {
  text: string;
  preview: BulkPastePreview | null;
};

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
  const [applyingAll, setApplyingAll] = useState(false);
  const [applyingTotal, setApplyingTotal] = useState(0);
  const [bulkByStage, setBulkByStage] = useState<Record<string, StageBulkState>>(
    {},
  );

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
  const singleImportInFlight =
    !applyingAll && Object.values(slotImporting).some(Boolean);
  const pageFrozen = applyingAll || singleImportInFlight;

  useEffect(() => {
    if (!pageFrozen) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [pageFrozen]);
  const pendingSlotIds = allSlots
    .filter(
      (sl) =>
        sl.beatmap === null &&
        !slotImporting[sl.id] &&
        (slotInputs[sl.id] ?? "").trim()
    )
    .map((sl) => sl.id);

  const importAndAssign = async (
    slotId: string,
    opts?: { skipRefresh?: boolean; rawInput?: string },
  ) => {
    const url = (opts?.rawInput ?? slotInputs[slotId] ?? "").trim();
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
      if (!opts?.skipRefresh) await refresh();
    } catch (e) {
      setSlotErrors((prev) => ({
        ...prev,
        [slotId]: e instanceof Error ? e.message : "Import failed",
      }));
    } finally {
      setSlotImporting((prev) => ({ ...prev, [slotId]: false }));
    }
  };

  const getBulkState = (stageId: string): StageBulkState =>
    bulkByStage[stageId] ?? { text: "", preview: null };

  const setBulkText = (stageId: string, text: string) => {
    setBulkByStage((prev) => ({
      ...prev,
      [stageId]: { text, preview: null },
    }));
  };

  const runBulkPreview = (stageId: string) => {
    const stage = tournament?.stages.find((s) => s.id === stageId);
    if (!stage) return;
    const text = getBulkState(stageId).text;
    setBulkByStage((prev) => ({
      ...prev,
      [stageId]: { text, preview: previewBulkPaste(text, stage) },
    }));
  };

  const APPLY_ALL_CONCURRENCY = 3;

  const applyBulkAssignments = async (
    stageId: string,
    assignments: BulkPasteAssignment[],
  ) => {
    if (assignments.length === 0) return;
    setApplyingAll(true);
    setApplyingTotal(assignments.length);
    setSlotInputs((prev) => {
      const next = { ...prev };
      for (const a of assignments) next[a.slotId] = a.rawInput;
      return next;
    });
    const queue = [...assignments];
    const worker = async () => {
      let next: BulkPasteAssignment | undefined;
      while ((next = queue.shift()) !== undefined) {
        await importAndAssign(next.slotId, {
          skipRefresh: true,
          rawInput: next.rawInput,
        });
      }
    };
    await Promise.allSettled(
      Array.from(
        { length: Math.min(APPLY_ALL_CONCURRENCY, queue.length) },
        worker,
      ),
    );
    await refresh();
    setBulkByStage((prev) => ({
      ...prev,
      [stageId]: { text: "", preview: null },
    }));
    setApplyingAll(false);
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

  const applyAll = async () => {
    if (pendingSlotIds.length === 0) return;
    setApplyingAll(true);
    setApplyingTotal(pendingSlotIds.length);
    const queue = [...pendingSlotIds];
    const worker = async () => {
      let slotId: string | undefined;
      while ((slotId = queue.shift()) !== undefined) {
        await importAndAssign(slotId, { skipRefresh: true });
      }
    };
    await Promise.allSettled(
      Array.from({ length: Math.min(APPLY_ALL_CONCURRENCY, queue.length) }, worker)
    );
    await refresh();
    setApplyingAll(false);
  };

  const runAnalysis = () => {
    setRunning(true);
    router.push(`/tournaments/${id}/report`);
  };

  if (error && !tournament) {
    return (
      <main className="programme">
        <div className="alert" role="alert">
          <span className="alert-icon" aria-hidden="true">▲</span>
          <p className="alert-text">Error: {error}</p>
        </div>
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
        <h1 className="masthead-title">Import Beatmaps</h1>
      </div>

      {error && (
        <div className="alert" role="alert">
          <span className="alert-icon" aria-hidden="true">▲</span>
          <p className="alert-text">{error}</p>
        </div>
      )}

      <div
        aria-hidden={pageFrozen || undefined}
        style={pageFrozen ? { pointerEvents: "none" } : undefined}
      >
        {tournament.stages.map((stage) => {
          const bulk = getBulkState(stage.id);
          const totalSlots = stageTotalSlots(stage);
          const categorySummary = stage.categories
            .map((c) => `${c.name}(${c.slots.length})`)
            .join(", ");
          return (
          <section key={stage.id} className="pool-stage">
            <div className="pool-stage-head">
              <div className="stage-head-title">
                <h2 className="stage-name">{stage.name}</h2>
                {stage.projectedStarRating != null && (
                  <span className="stage-meta">
                    ★ {stage.projectedStarRating.toFixed(2)} projected
                  </span>
                )}
              </div>
            </div>

            <div className="pool-bulk-paste">
              <label className="pool-bulk-label" htmlFor={`bulk-${stage.id}`}>
                Bulk paste beatmap IDs
                <span className="pool-bulk-hint">
                  {totalSlots} slots · {categorySummary || "no categories"} · one
                  ID per line, in category order
                </span>
              </label>
              <textarea
                id={`bulk-${stage.id}`}
                className="field-input pool-bulk-textarea"
                rows={Math.min(8, Math.max(4, totalSlots))}
                placeholder={"880321\n975588\n2893305\n…"}
                value={bulk.text}
                onChange={(e) => setBulkText(stage.id, e.target.value)}
                disabled={pageFrozen}
                aria-describedby={`bulk-help-${stage.id}`}
              />
              <p id={`bulk-help-${stage.id}`} className="pool-bulk-help">
                Paste fills slots in this stage&apos;s category order (e.g. NM1…NMn,
                then HD1…, and so on). Count must match exactly.
              </p>
              <div className="pool-bulk-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => runBulkPreview(stage.id)}
                  disabled={pageFrozen || !bulk.text.trim()}
                >
                  Preview mapping
                </button>
                {bulk.preview?.ok === true && (
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={() => {
                      const preview = getBulkState(stage.id).preview;
                      if (preview?.ok) {
                        void applyBulkAssignments(stage.id, preview.assignments);
                      }
                    }}
                    disabled={pageFrozen}
                  >
                    Apply {bulk.preview.assignments.length} to {stage.name}
                  </button>
                )}
              </div>

              {bulk.preview && !bulk.preview.ok && (
                <div className="pool-bulk-feedback" role="alert">
                  {bulk.preview.kind === "empty" && (
                    <p className="slot-error">
                      ▲ Paste at least one beatmap ID (one per line).
                    </p>
                  )}
                  {bulk.preview.kind === "count_mismatch" && (
                    <p className="slot-error">
                      ▲ Expected {bulk.preview.expected} IDs for this stage, got{" "}
                      {bulk.preview.got}. No slots were changed.
                    </p>
                  )}
                  {bulk.preview.kind === "malformed" && (
                    <ul className="pool-bulk-errors">
                      {bulk.preview.errors.map((err) => (
                        <li key={`${err.line}-${err.raw}`} className="slot-error">
                          ▲ Line {err.line}: {err.message}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}

              {bulk.preview?.ok && (
                <div
                  className="pool-bulk-preview"
                  aria-label={`Preview assignment for ${stage.name}`}
                >
                  <p className="pool-bulk-preview-title">
                    Confirm mapping ({bulk.preview.totalSlots} slots)
                  </p>
                  <ol className="pool-bulk-preview-list">
                    {bulk.preview.assignments.map((a) => (
                      <li key={a.slotId}>
                        <span className="pool-bulk-slot">{a.slotCode}</span>
                        <span className="pool-bulk-arrow" aria-hidden="true">
                          →
                        </span>
                        <span className="pool-bulk-id">{a.beatmapId}</span>
                      </li>
                    ))}
                  </ol>
                </div>
              )}
            </div>

            {stage.categories.map((cat) => (
              <div key={cat.id} className="pool-category-block">
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
                        className={`slot-row slot-row--editable${hasCover ? " slot-row--cover" : ""}${
                          slotImporting[slot.id] ? " slot-row--loading" : ""
                        }`}
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
                              {formatStarRating(slot.effectiveDifficulty?.starRating)} ·{" "}
                              {formatBpm(slot.beatmap.bpm, slot.effectiveDifficulty?.bpm)} ·{" "}
                              {formatStat("CS", slot.beatmap.cs, slot.effectiveDifficulty?.cs)} ·{" "}
                              {formatStat("AR", slot.beatmap.ar, slot.effectiveDifficulty?.ar)} ·{" "}
                              {formatStat("OD", slot.beatmap.od, slot.effectiveDifficulty?.od)}
                            </span>
                            <button
                              className="btn btn-ghost pool-slot-clear"
                              onClick={() => clear(slot.id)}
                              title="Clear slot"
                              aria-label={`Clear beatmap from slot ${slot.code}`}
                            >
                              <Trash2 size={14} aria-hidden="true" />
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
                              {slotImporting[slot.id] ? (
                                <span className="spinner" aria-hidden="true" />
                              ) : (
                                <ClipboardCheck size={14} aria-hidden="true" />
                              )}
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
          );
        })}
      </div>

      {(pendingSlotIds.length > 0 || applyingAll) && (
        <div className="pool-apply-all">
          <button
            className="btn btn-ghost"
            onClick={applyAll}
            disabled={applyingAll}
            title="Import & assign all pasted beatmaps"
          >
            {applyingAll ? (
              <>
                <span className="spinner" aria-hidden="true" /> Importing…
              </>
            ) : (
              `Import All (${pendingSlotIds.length})`
            )}
          </button>
        </div>
      )}

      <div
        className="wizard-nav"
        aria-hidden={pageFrozen || undefined}
        style={pageFrozen ? { pointerEvents: "none" } : undefined}
      >
        <span className="wizard-step-indicator">
          {filledCount} / {totalCount} slots filled
          {filledCount < totalCount && (
            <span style={{ color: "var(--mark)" }}>
              {" "}· {totalCount - filledCount} slot
              {totalCount - filledCount !== 1 ? "s" : ""} still need a beatmap
            </span>
          )}
        </span>
        <button
          className="btn btn-primary"
          onClick={runAnalysis}
          disabled={running || totalCount === 0 || filledCount < totalCount}
        >
          {running ? "Running Analysis…" : "Run Analysis →"}
        </button>
      </div>

      {(applyingAll || singleImportInFlight) && (
        <div className="loading-overlay" role="status" aria-live="polite">
          <span className="spinner spinner--lg" aria-hidden="true" />
          <span className="loading-overlay-text">
            {applyingAll
              ? `Importing ${applyingTotal} beatmap${applyingTotal === 1 ? "" : "s"}…`
              : "Importing beatmap…"}
          </span>
        </div>
      )}
    </main>
  );
}
