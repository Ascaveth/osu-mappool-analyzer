"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import type { Tournament, Beatmap } from "@/lib/types";

export default function PoolPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();

  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [beatmaps, setBeatmaps] = useState<Beatmap[]>([]);
  const [selectedSlotId, setSelectedSlotId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const [t, bms] = await Promise.all([
        api.getTournament(id),
        api.listBeatmaps(),
      ]);
      setTournament(t);
      setBeatmaps(bms);
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

  const filteredBeatmaps = beatmaps.filter((bm) => {
    if (!search.trim()) return true;
    const q = search.toLowerCase();
    return (
      bm.title.toLowerCase().includes(q) ||
      bm.artist.toLowerCase().includes(q) ||
      bm.mapper.toLowerCase().includes(q) ||
      bm.version.toLowerCase().includes(q)
    );
  });

  const assign = async (beatmapId: string) => {
    if (!selectedSlotId) return;
    try {
      await api.assignBeatmap(selectedSlotId, beatmapId);
      setSelectedSlotId(null);
      await refresh();
    } catch (e) {
      setSelectedSlotId(null);
      setError(e instanceof Error ? e.message : "Failed to assign beatmap");
    }
  };

  const clear = async (slotId: string) => {
    try {
      await api.clearBeatmap(slotId);
      if (selectedSlotId === slotId) setSelectedSlotId(null);
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
          href={`/tournaments/${id}/import`}
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
            href={`/tournaments/${id}/import`}
            style={{ color: "inherit", textDecoration: "none" }}
          >
            ← Back
          </Link>
          {" · "}Step 3 of 3{" · "}
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

      <div className="pool-editor">
        {/* Left: slot grid */}
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
                  <p className="category-name">{cat.name}</p>
                  {cat.slots.map((slot) => (
                    <div
                      key={slot.id}
                      role="button"
                      tabIndex={0}
                      className={`slot-row pool-slot${selectedSlotId === slot.id ? " pool-slot--selected" : ""}`}
                      onClick={() =>
                        setSelectedSlotId(
                          slot.id === selectedSlotId ? null : slot.id,
                        )
                      }
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          setSelectedSlotId(
                            slot.id === selectedSlotId ? null : slot.id,
                          );
                        }
                      }}
                    >
                      <span className="slot-code">{slot.code}</span>

                      {slot.beatmap ? (
                        <>
                          <span>
                            <span className="slot-title">
                              {slot.beatmap.title}
                            </span>{" "}
                            <span className="slot-artist">
                              — {slot.beatmap.artist}
                            </span>
                          </span>
                          <span className="slot-stats">
                            AR {slot.beatmap.ar.toFixed(1)} · OD{" "}
                            {slot.beatmap.od.toFixed(1)} · {slot.beatmap.bpm}{" "}
                            BPM
                          </span>
                          <button
                            className="btn btn-ghost pool-slot-clear"
                            onClick={(e) => {
                              e.stopPropagation();
                              clear(slot.id);
                            }}
                            title="Clear slot"
                          >
                            ×
                          </button>
                        </>
                      ) : (
                        <span className="pool-slot-empty">
                          {selectedSlotId === slot.id
                            ? "← pick a beatmap from the library"
                            : "click to select"}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              ))}
            </section>
          ))}
        </div>

        {/* Right: beatmap library */}
        <aside className="pool-library">
          <p className="category-name" style={{ marginBottom: "0.5rem" }}>
            Beatmap Library
          </p>
          <input
            className="field-input"
            placeholder="Search title, artist, mapper…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ marginBottom: "0.75rem" }}
          />
          {beatmaps.length === 0 ? (
            <p className="slot-stats" style={{ marginTop: "0.5rem" }}>
              No beatmaps imported yet.{" "}
              <Link
                href={`/tournaments/${id}/import`}
                style={{ color: "var(--brass)" }}
              >
                Go back to Import.
              </Link>
            </p>
          ) : filteredBeatmaps.length === 0 ? (
            <p className="slot-stats">No matches.</p>
          ) : (
            filteredBeatmaps.map((bm) => (
              <div
                key={bm.id}
                role={selectedSlotId ? "button" : undefined}
                tabIndex={selectedSlotId ? 0 : undefined}
                className={`bm-card${selectedSlotId ? " bm-card--clickable" : ""}`}
                onClick={() => selectedSlotId && assign(bm.id)}
                onKeyDown={(e) => {
                  if (selectedSlotId && (e.key === "Enter" || e.key === " ")) {
                    e.preventDefault();
                    assign(bm.id);
                  }
                }}
              >
                <p className="bm-card-title">{bm.title}</p>
                <p className="bm-card-meta">
                  {bm.version} · {bm.mapper}
                </p>
                <p className="bm-card-meta">
                  AR {bm.ar.toFixed(1)} · OD {bm.od.toFixed(1)} · {bm.bpm} BPM
                </p>
              </div>
            ))
          )}
        </aside>
      </div>

      <div className="wizard-nav">
        <span className="wizard-step-indicator">
          {filledCount} / {totalCount} slots filled
          {selectedSlotId ? " · slot selected — click a beatmap to assign" : ""}
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
