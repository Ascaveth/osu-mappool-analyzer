"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createMockClient } from "@/lib/api";
import type { Beatmap, Tournament } from "@/lib/types";

const api = createMockClient();

export default function ImportPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();

  const [tournament, setTournament] = useState<Tournament | null>(null);
  const [beatmaps, setBeatmaps] = useState<Beatmap[]>([]);
  const [urlInput, setUrlInput] = useState("");
  const [importing, setImporting] = useState(false);
  const [errors, setErrors] = useState<string[]>([]);

  useEffect(() => {
    api.getTournament(id).then(setTournament).catch(console.error);
    api.listBeatmaps().then(setBeatmaps).catch(console.error);
  }, [id]);

  const importUrls = useCallback(async (raw: string) => {
    const urls = raw
      .split(/[\n,]+/)
      .map((u) => u.trim())
      .filter(Boolean);
    if (urls.length === 0) return;
    setImporting(true);
    const errs: string[] = [];
    for (const url of urls) {
      try {
        const bm = await api.importBeatmapFromUrl(url);
        setBeatmaps((prev) =>
          prev.some((b) => b.id === bm.id) ? prev : [...prev, bm],
        );
      } catch (e) {
        errs.push(`${url}: ${e instanceof Error ? e.message : "import error"}`);
      }
    }
    setErrors(errs);
    setImporting(false);
  }, []);

  const handleAdd = () => {
    importUrls(urlInput);
    setUrlInput("");
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") handleAdd();
  };

  const totalSlots = tournament
    ? tournament.stages.reduce(
        (a, s) => a + s.categories.reduce((b, c) => b + c.slots.length, 0),
        0,
      )
    : 0;

  return (
    <main className="programme">
      <div className="masthead">
        <p className="masthead-eyebrow">
          <Link href="/tournaments/new" style={{ color: "inherit", textDecoration: "none" }}>
            ← Back
          </Link>
          {" · "}Step 2 of 3{" · "}
          {tournament?.name ?? "…"}
        </p>
        <h1 className="masthead-title">Import Beatmaps</h1>
      </div>

      <p
        style={{
          fontFamily: "var(--font-body)",
          fontSize: "0.9375rem",
          color: "var(--ink-soft)",
          marginBottom: "1.25rem",
          lineHeight: 1.5,
        }}
      >
        Paste osu! beatmap URLs to import. Accepts{" "}
        <span style={{ fontFamily: "var(--font-data)", fontSize: "0.8125rem" }}>
          osu.ppy.sh/beatmapsets/…#osu/…
        </span>{" "}
        links. Multiple URLs separated by newlines or commas.
      </p>

      <div style={{ display: "flex", gap: "0.5rem", alignItems: "stretch" }}>
        <input
          className="field-input"
          style={{ flex: 1 }}
          placeholder="https://osu.ppy.sh/beatmapsets/1555041#osu/3176982"
          value={urlInput}
          onChange={(e) => setUrlInput(e.target.value)}
          onKeyDown={onKeyDown}
          disabled={importing}
        />
        <button
          className="btn btn-primary"
          onClick={handleAdd}
          disabled={importing || !urlInput.trim()}
        >
          {importing ? "Importing…" : "Add"}
        </button>
      </div>

      {errors.length > 0 && (
        <div style={{ marginTop: "0.75rem" }}>
          {errors.map((err, i) => (
            <p
              key={i}
              style={{
                color: "var(--mark)",
                fontFamily: "var(--font-data)",
                fontSize: "0.75rem",
                lineHeight: 1.5,
              }}
            >
              ▲ {err}
            </p>
          ))}
        </div>
      )}

      {beatmaps.length > 0 && (
        <div style={{ marginTop: "2rem" }}>
          <p className="category-name" style={{ marginBottom: "0.75rem" }}>
            Imported · {beatmaps.length} beatmap{beatmaps.length !== 1 ? "s" : ""}
          </p>
          {beatmaps.map((bm) => (
            <div key={bm.id} className="bm-card">
              <p className="bm-card-title">
                {bm.title}{" "}
                <span style={{ color: "var(--ink-soft)", fontSize: "0.8125rem" }}>
                  — {bm.artist}
                </span>
              </p>
              <p className="bm-card-meta">
                {bm.version} · {bm.mapper} · AR {bm.ar.toFixed(1)} · OD{" "}
                {bm.od.toFixed(1)} · {bm.bpm} BPM
              </p>
            </div>
          ))}
        </div>
      )}

      <div className="wizard-nav">
        <span className="wizard-step-indicator">
          {beatmaps.length} imported · {totalSlots} slot{totalSlots !== 1 ? "s" : ""} to fill
        </span>
        <button
          className="btn btn-primary"
          onClick={() => router.push(`/tournaments/${id}/pool`)}
        >
          Continue to Pool Builder →
        </button>
      </div>
    </main>
  );
}
