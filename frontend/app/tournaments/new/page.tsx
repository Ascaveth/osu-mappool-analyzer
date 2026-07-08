"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { CreateStageInput, CreateCategoryInput } from "@/lib/api";
import { modAccentColor } from "@/lib/beatmap-format";
import { KNOWN_MODS } from "@/lib/mods";

interface CatDraft {
  _id: string;
  modPrefix: string;
  slotCount: number;
}

interface StageDraft {
  _id: string;
  name: string;
  categories: CatDraft[];
  projectedStarRating: string;
}

interface ProofNote {
  key: string;
  kind: "warn" | "ready";
  text: string;
  source: string;
}

function draftId() {
  return Math.random().toString(36).slice(2);
}

function newCat(): CatDraft {
  return { _id: draftId(), modPrefix: "NM", slotCount: 2 };
}

function newStage(): StageDraft {
  return { _id: draftId(), name: "", categories: [newCat()], projectedStarRating: "" };
}

function hasDuplicateMods(categories: CatDraft[]): boolean {
  if (categories.some((c) => c.modPrefix.trim().length === 0)) return true;
  return new Set(categories.map((c) => c.modPrefix)).size !== categories.length;
}

// A blank Star Rating is valid (unset — falls back to NM1). Otherwise it
// must be a plain non-negative decimal, e.g. "6.75" — no letters, no
// second decimal separator, nothing left over after normalizing a
// comma-as-decimal input like "6,7" to "6.7".
function isValidStarRating(raw: string): boolean {
  const trimmed = raw.trim();
  return trimmed === "" || /^\d+(\.\d+)?$/.test(trimmed);
}

// Accepts either "." or "," as the decimal separator (some locales/keyboards
// only offer ","), normalizing to "." for storage/parsing. Anything that
// isn't digits plus at most one separator is rejected outright rather than
// silently truncated, so a stray letter never ends up "swallowed" into a
// seemingly-valid number.
function normalizeStarRatingInput(raw: string): string | null {
  const normalized = raw.replace(",", ".");
  return /^\d*\.?\d*$/.test(normalized) ? normalized : null;
}

function slotCodes(cat: CatDraft): string[] {
  return Array.from({ length: cat.slotCount }, (_, i) => `${cat.modPrefix}${i + 1}`);
}

export default function NewTournamentPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [stages, setStages] = useState<StageDraft[]>([newStage()]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateStage = (sid: string, patch: Partial<StageDraft>) =>
    setStages((prev) => prev.map((s) => (s._id === sid ? { ...s, ...patch } : s)));

  const updateCat = (sid: string, cid: string, patch: Partial<CatDraft>) =>
    setStages((prev) =>
      prev.map((s) =>
        s._id === sid
          ? {
              ...s,
              categories: s.categories.map((c) =>
                c._id === cid ? { ...c, ...patch } : c,
              ),
            }
          : s,
      ),
    );

  const addStage = () => setStages((prev) => [...prev, newStage()]);
  const removeStage = (sid: string) =>
    setStages((prev) => prev.filter((s) => s._id !== sid));

  const addCat = (sid: string) =>
    setStages((prev) =>
      prev.map((s) =>
        s._id === sid ? { ...s, categories: [...s.categories, newCat()] } : s,
      ),
    );
  const removeCat = (sid: string, cid: string) =>
    setStages((prev) =>
      prev.map((s) =>
        s._id === sid
          ? { ...s, categories: s.categories.filter((c) => c._id !== cid) }
          : s,
      ),
    );

  const onModChange = (sid: string, cid: string, rawModPrefix: string) => {
    const modPrefix = rawModPrefix.toUpperCase();
    const patch: Partial<CatDraft> = { modPrefix };
    if (modPrefix === "TB") patch.slotCount = 1;
    updateCat(sid, cid, patch);
  };

  const totalSlots = stages.reduce(
    (a, s) => a + s.categories.reduce((b, c) => b + c.slotCount, 0),
    0,
  );

  const valid =
    name.trim().length > 0 &&
    stages.length > 0 &&
    stages.every(
      (s) =>
        s.name.trim().length > 0 &&
        s.categories.length > 0 &&
        s.categories.every(
          (c) =>
            c.slotCount >= 1 &&
            c.slotCount <= 20 &&
            (c.modPrefix !== "TB" || c.slotCount === 1),
        ) &&
        !hasDuplicateMods(s.categories) &&
        isValidStarRating(s.projectedStarRating),
    );

  const notes: ProofNote[] = [];
  if (!name.trim()) {
    notes.push({
      key: "name",
      kind: "warn",
      text: "The tournament needs a name before it can run.",
      source: "Title",
    });
  }
  stages.forEach((s, si) => {
    const label = s.name.trim() || `Stage ${si + 1}`;
    if (!s.name.trim()) {
      notes.push({
        key: `${s._id}-name`,
        kind: "warn",
        text: `Stage ${si + 1} needs a name.`,
        source: label,
      });
    }
    if (hasDuplicateMods(s.categories)) {
      notes.push({
        key: `${s._id}-dup`,
        kind: "warn",
        text: "Two categories share a mod — give each a distinct one.",
        source: label,
      });
    }
    if (!isValidStarRating(s.projectedStarRating)) {
      notes.push({
        key: `${s._id}-sr`,
        kind: "warn",
        text: "Projected Star Rating must be a plain decimal number, e.g. 6.75.",
        source: label,
      });
    }
  });
  if (valid) {
    notes.push({
      key: "ready",
      kind: "ready",
      text: "Everything looks good!",
      source: "",
    });
  }

  const handleSubmit = async () => {
    if (!valid || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const input = {
        name: name.trim(),
        stages: stages.map((s, si): CreateStageInput => ({
          name: s.name.trim(),
          order: si + 1,
          categories: s.categories.map((c, ci): CreateCategoryInput => ({
            order: ci + 1,
            modPrefix: c.modPrefix,
            slotCount: c.slotCount,
          })),
          projectedStarRating: s.projectedStarRating.trim()
            ? Number(s.projectedStarRating)
            : undefined,
        })),
      };
      const t = await api.createTournament(input);
      router.push(`/tournaments/${t.id}/pool`);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      setSubmitting(false);
    }
  };

  return (
    <main className="programme">
      <div className="masthead">
        <p className="masthead-eyebrow">Step 1 of 2 · Tournament Setup</p>
        <h1 className="masthead-title">Tournament Setup</h1>
      </div>

      {/* Suggestions only — tournament structures are user-defined, so any
          mod prefix typed into the field above is accepted as-is. */}
      <datalist id="mod-category-options">
        {KNOWN_MODS.map((m) => (
          <option key={m.value} value={m.value}>
            {m.label}
          </option>
        ))}
      </datalist>

      <div className="setup-layout">
        <div>
          <div className="field">
            <label className="field-label" htmlFor="t-name">
              Tournament Name
            </label>
            <input
              id="t-name"
              className="field-input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ascaveth Invitational 2023"
            />
          </div>

          <div style={{ marginTop: "2rem" }}>
            <p className="category-name" style={{ marginBottom: "1rem" }}>
              Stages
            </p>

            {stages.map((stage, si) => (
              <div key={stage._id} className="stage-builder-item">
                <div className="stage-builder-header">
                  <input
                    className="field-input"
                    value={stage.name}
                    onChange={(e) => updateStage(stage._id, { name: e.target.value })}
                    placeholder="Stage name (e.g. Qualifiers, Round of 16, Grand Finals)"
                    style={{ flex: 1 }}
                  />
                  <div className="inline-field">
                    <input
                      className="field-input"
                      type="text"
                      inputMode="decimal"
                      placeholder="e.g. 6.75"
                      aria-label="Projected star rating for this stage (optional)"
                      aria-invalid={!isValidStarRating(stage.projectedStarRating)}
                      title="Optional target Star Rating. Defaults to this stage's NM1 beatmap once one is assigned."
                      value={stage.projectedStarRating}
                      onChange={(e) => {
                        const normalized = normalizeStarRatingInput(e.target.value);
                        if (normalized !== null) {
                          updateStage(stage._id, { projectedStarRating: normalized });
                        }
                      }}
                      style={{
                        width: "4.5rem",
                        borderColor: isValidStarRating(stage.projectedStarRating)
                          ? undefined
                          : "var(--mark)",
                      }}
                    />
                    <span className="slot-stats">★ SR</span>
                  </div>
                  {stages.length > 1 && (
                    <button
                      className="btn btn-ghost"
                      onClick={() => removeStage(stage._id)}
                    >
                      Remove
                    </button>
                  )}
                </div>

                <div className="cat-list">
                  <p className="category-name" style={{ marginBottom: "0.4rem" }}>
                    Categories
                  </p>
                  {stage.categories.map((cat) => {
                    const isDupMod =
                      stage.categories.filter((c) => c.modPrefix === cat.modPrefix)
                        .length > 1;
                    const isTB = cat.modPrefix === "TB";
                    const dotColor = modAccentColor(`${cat.modPrefix}1`);
                    return (
                      <div key={cat._id} className="cat-builder-row">
                        <div className="mod-select-wrap">
                          {dotColor && (
                            <span
                              className="category-dot"
                              style={{ background: dotColor }}
                            />
                          )}
                          <input
                            className="field-input"
                            list="mod-category-options"
                            aria-label="Mod category"
                            placeholder="NM"
                            value={cat.modPrefix}
                            onChange={(e) =>
                              onModChange(stage._id, cat._id, e.target.value)
                            }
                            style={{
                              borderColor: isDupMod ? "var(--mark)" : undefined,
                            }}
                          />
                        </div>
                        <div className="inline-field">
                          <input
                            className="field-input"
                            type="number"
                            aria-label="Number of slots"
                            min={1}
                            max={20}
                            value={cat.slotCount}
                            disabled={isTB}
                            onChange={(e) =>
                              updateCat(stage._id, cat._id, {
                                slotCount: isTB
                                  ? 1
                                  : Math.min(
                                      20,
                                      Math.max(1, parseInt(e.target.value) || 1),
                                    ),
                              })
                            }
                            style={{ width: "3.5rem" }}
                            title={isTB ? "Tiebreaker is locked to 1 slot" : "Number of slots"}
                          />
                          <span className="slot-stats">slots</span>
                        </div>
                        {stage.categories.length > 1 && (
                          <button
                            className="btn btn-ghost"
                            aria-label="Remove category"
                            style={{ flex: "none", padding: "0.25rem 0.5rem" }}
                            onClick={() => removeCat(stage._id, cat._id)}
                            title="Remove category"
                          >
                            ×
                          </button>
                        )}
                      </div>
                    );
                  })}
                  <button
                    className="btn btn-ghost"
                    style={{ marginTop: "0.4rem", fontSize: "0.6875rem" }}
                    onClick={() => addCat(stage._id)}
                  >
                    + Add Category
                  </button>
                </div>
              </div>
            ))}

            <button className="btn btn-ghost" onClick={addStage} style={{ marginTop: "0.5rem" }}>
              + Add Stage
            </button>
          </div>

          {error && (
            <div className="alert" role="alert">
              <span className="alert-icon" aria-hidden="true">▲</span>
              <p className="alert-text">{error}</p>
            </div>
          )}
        </div>

        <aside className="proof-panel" aria-label="Setup checks">
          <p className="proof-eyebrow">
            <span>Checks before you continue</span>
          </p>
          {notes.length === 0 ? (
            <p className="proof-empty">Add a stage to see checks here.</p>
          ) : (
            <div className="proof-notes">
              {notes.map((n) => (
                <div className="note" key={n.key}>
                  <span
                    className={`note-mark note-mark--${n.kind === "ready" ? "ready" : "warning"}`}
                    aria-hidden="true"
                  >
                    {n.kind === "ready" ? "✓" : "▲"}
                  </span>
                  <div className="note-body">
                    <p className="note-text">{n.text}</p>
                    <span className="note-source">{n.source}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </aside>
      </div>

      <div className="wizard-nav">
        <span className="wizard-step-indicator">
          {stages.length} stage{stages.length !== 1 ? "s" : ""} · {totalSlots} total slot{totalSlots !== 1 ? "s" : ""}
        </span>
        <button
          className="btn btn-primary"
          onClick={handleSubmit}
          disabled={!valid || submitting}
        >
          {submitting ? "Creating…" : "Continue to Pool →"}
        </button>
      </div>
    </main>
  );
}
