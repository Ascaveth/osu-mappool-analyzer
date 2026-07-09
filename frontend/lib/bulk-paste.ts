import { extractBeatmapId } from "@/lib/beatmap-id";
import type { Category, Slot, Stage } from "@/lib/types";

export type BulkPasteLineError = {
  line: number;
  raw: string;
  message: string;
};

export type BulkPasteAssignment = {
  slotId: string;
  slotCode: string;
  categoryName: string;
  rawInput: string;
  beatmapId: string;
};

export type BulkPastePreview =
  | { ok: true; assignments: BulkPasteAssignment[]; totalSlots: number }
  | { ok: false; kind: "count_mismatch"; expected: number; got: number }
  | { ok: false; kind: "malformed"; errors: BulkPasteLineError[] }
  | { ok: false; kind: "empty" };

/** Non-empty trimmed lines with 1-based source line numbers. */
export function parseBulkPasteLines(
  text: string,
): { line: number; raw: string }[] {
  return text
    .split(/\r?\n/)
    .map((raw, i) => ({ line: i + 1, raw: raw.trim() }))
    .filter((entry) => entry.raw.length > 0);
}

/**
 * Slots in stage display order: categories as stored, then slots within each.
 * Matches the pool builder's render order (not alphabetical / ID-based).
 */
export function orderedSlotsForStage(
  stage: Stage,
): { slot: Slot; category: Category }[] {
  return stage.categories.flatMap((category) =>
    category.slots.map((slot) => ({ slot, category })),
  );
}

export function stageTotalSlots(stage: Stage): number {
  return stage.categories.reduce((n, c) => n + c.slots.length, 0);
}

/**
 * Preview a newline-separated paste against a stage's category/slot layout.
 * Does not mutate state or call the API — commit happens separately after confirm.
 */
export function previewBulkPaste(text: string, stage: Stage): BulkPastePreview {
  const lines = parseBulkPasteLines(text);
  const slots = orderedSlotsForStage(stage);
  const totalSlots = slots.length;

  if (lines.length === 0) {
    return { ok: false, kind: "empty" };
  }

  const errors: BulkPasteLineError[] = [];
  const beatmapIds: string[] = [];

  for (const { line, raw } of lines) {
    try {
      beatmapIds.push(extractBeatmapId(raw));
    } catch (e) {
      errors.push({
        line,
        raw,
        message: e instanceof Error ? e.message : `Invalid beatmap ID: ${raw}`,
      });
    }
  }

  if (errors.length > 0) {
    return { ok: false, kind: "malformed", errors };
  }

  if (lines.length !== totalSlots) {
    return {
      ok: false,
      kind: "count_mismatch",
      expected: totalSlots,
      got: lines.length,
    };
  }

  const assignments: BulkPasteAssignment[] = slots.map(({ slot, category }, i) => ({
    slotId: slot.id,
    slotCode: slot.code,
    categoryName: category.name,
    rawInput: lines[i].raw,
    beatmapId: beatmapIds[i],
  }));

  return { ok: true, assignments, totalSlots };
}
