import { describe, expect, it } from "vitest";
import { previewBulkPaste } from "@/lib/bulk-paste";
import type { Stage } from "@/lib/types";

function slot(id: string, code: string) {
  return { id, code, beatmap: null };
}

/** NM(2), HD(1), TB(1) — 4 slots in category display order. */
function sampleStage(): Stage {
  return {
    id: "stage-1",
    name: "Group Stage",
    order: 1,
    projectedStarRating: null,
    categories: [
      {
        id: "cat-nm",
        name: "NM",
        order: 1,
        slots: [slot("s-nm1", "NM1"), slot("s-nm2", "NM2")],
      },
      {
        id: "cat-hd",
        name: "HD",
        order: 2,
        slots: [slot("s-hd1", "HD1")],
      },
      {
        id: "cat-tb",
        name: "TB",
        order: 3,
        slots: [slot("s-tb1", "TB1")],
      },
    ],
  };
}

describe("previewBulkPaste", () => {
  it("assigns IDs in category/slot display order when count matches totalSlots", () => {
    const text = ["880321", "975588", "2893305", "3111046"].join("\n");
    const result = previewBulkPaste(text, sampleStage());

    expect(result.ok).toBe(true);
    if (!result.ok) return;

    expect(result.totalSlots).toBe(4);
    expect(result.assignments.map((a) => a.slotCode)).toEqual([
      "NM1",
      "NM2",
      "HD1",
      "TB1",
    ]);
    expect(result.assignments.map((a) => a.beatmapId)).toEqual([
      "880321",
      "975588",
      "2893305",
      "3111046",
    ]);
    expect(result.assignments.map((a) => a.categoryName)).toEqual([
      "NM",
      "NM",
      "HD",
      "TB",
    ]);
  });

  it("tolerates blank lines and surrounding whitespace", () => {
    const text = "\n  880321  \n\n975588\n2893305\n\n3111046\n";
    const result = previewBulkPaste(text, sampleStage());

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.assignments.map((a) => a.beatmapId)).toEqual([
      "880321",
      "975588",
      "2893305",
      "3111046",
    ]);
  });

  it("accepts beatmap URLs mixed with bare IDs", () => {
    const text = [
      "https://osu.ppy.sh/beatmaps/880321",
      "https://osu.ppy.sh/beatmapsets/1#osu/975588",
      "2893305",
      "3111046",
    ].join("\n");
    const result = previewBulkPaste(text, sampleStage());

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.assignments.map((a) => a.beatmapId)).toEqual([
      "880321",
      "975588",
      "2893305",
      "3111046",
    ]);
  });

  it("rejects too few IDs with a clear count mismatch (no partial assignment)", () => {
    const result = previewBulkPaste("880321\n975588", sampleStage());

    expect(result).toEqual({
      ok: false,
      kind: "count_mismatch",
      expected: 4,
      got: 2,
    });
  });

  it("rejects too many IDs with a clear count mismatch (no partial assignment)", () => {
    const text = ["1", "2", "3", "4", "5"].join("\n");
    const result = previewBulkPaste(text, sampleStage());

    expect(result).toEqual({
      ok: false,
      kind: "count_mismatch",
      expected: 4,
      got: 5,
    });
  });

  it("rejects non-numeric or malformed lines before commit", () => {
    const text = ["880321", "not-a-url", "2893305", "abc"].join("\n");
    const result = previewBulkPaste(text, sampleStage());

    expect(result.ok).toBe(false);
    if (result.ok || result.kind !== "malformed") {
      expect.fail("expected malformed preview");
      return;
    }

    expect(result.errors).toHaveLength(2);
    expect(result.errors[0]).toMatchObject({ line: 2, raw: "not-a-url" });
    expect(result.errors[1]).toMatchObject({ line: 4, raw: "abc" });
    expect(result.errors[0].message).toContain("Cannot parse beatmap ID");
  });

  it("reports empty paste distinctly", () => {
    expect(previewBulkPaste("  \n\n  ", sampleStage())).toEqual({
      ok: false,
      kind: "empty",
    });
  });

  it("validates malformed lines before count mismatch", () => {
    // 2 lines for a 4-slot stage, but one is malformed — malformed wins.
    const result = previewBulkPaste("880321\nbad", sampleStage());
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("malformed");
  });
});
