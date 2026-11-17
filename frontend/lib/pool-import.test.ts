import { describe, expect, it, vi } from "vitest";
import { importPoolBeatmaps, type PoolImportUpdate } from "./pool-import";
import type { Beatmap, Tournament } from "./types";

const tournament: Tournament = { id: "cup", name: "Cup", edition: "", stages: [] };
const beatmap: Beatmap = {
  id: "map", title: "Map", artist: "Artist", mapper: "Mapper", version: "Hard",
  ar: 9, od: 8, cs: 4, hp: 5, bpm: 180,
};

function clientFixture() {
  return {
    importBeatmapFromUrl: vi.fn(async () => beatmap),
    assignBeatmap: vi.fn(async (slotId: string, beatmapId: string) => ({
      id: slotId, code: "", beatmap: { ...beatmap, id: beatmapId },
    })),
    getTournament: vi.fn(async () => tournament),
  };
}

describe("Pool import", () => {
  it("imports before assigning and returns the authoritative Tournament after assignment", async () => {
    const client = clientFixture();
    const steps: string[] = [];
    client.importBeatmapFromUrl.mockImplementation(async () => {
      steps.push("import");
      return beatmap;
    });
    client.assignBeatmap.mockImplementation(async (slotId) => {
      steps.push("assign");
      return { id: slotId, code: "", beatmap };
    });
    client.getTournament.mockImplementation(async () => {
      steps.push("refresh");
      return tournament;
    });
    const updates: PoolImportUpdate[] = [];

    const result = await importPoolBeatmaps({
      client, tournamentId: "cup", assignments: [{ slotId: "slot", rawInput: " 3176982 " }],
      onUpdate: (update) => updates.push(update),
    });

    expect(result).toBe(tournament);
    expect(steps).toEqual(["import", "assign", "refresh"]);
    expect(client.importBeatmapFromUrl).toHaveBeenCalledWith("3176982");
    expect(client.assignBeatmap).toHaveBeenCalledWith("slot", "map");
    expect(client.getTournament).toHaveBeenCalledExactlyOnceWith("cup");
    expect(updates).toEqual([
      { slotId: "slot", kind: "importing" }, { slotId: "slot", kind: "assigned" },
    ]);
  });

  it("continues after import and assignment failures and reports each Slot outcome", async () => {
    const client = clientFixture();
    client.importBeatmapFromUrl.mockRejectedValueOnce(new Error("Provider unavailable"));
    client.assignBeatmap.mockRejectedValueOnce(new Error("Assignment failed"));
    const updates: PoolImportUpdate[] = [];

    await importPoolBeatmaps({
      client, tournamentId: "cup",
      assignments: ["a", "b", "c", "d"].map((slotId) => ({ slotId, rawInput: "3176982" })),
      onUpdate: (update) => updates.push(update),
    });

    expect(client.importBeatmapFromUrl).toHaveBeenCalledTimes(4);
    expect(client.assignBeatmap).toHaveBeenCalledTimes(3);
    expect(client.assignBeatmap).not.toHaveBeenCalledWith("a", expect.anything());
    expect(updates.filter((update) => update.kind === "assigned").map((update) => update.slotId)).toEqual(["c", "d"]);
    expect(updates).toContainEqual({ slotId: "a", kind: "failed", message: expect.stringContaining("Provider unavailable") });
    expect(updates).toContainEqual({ slotId: "b", kind: "failed", message: expect.stringContaining("Assignment failed") });
    expect(client.getTournament).toHaveBeenCalledTimes(1);
  });

  it("keeps at most three assignments in flight and refreshes after the final assignment", async () => {
    const client = clientFixture();
    const release = new Map<string, () => void>();
    let active = 0;
    let maximum = 0;
    client.importBeatmapFromUrl.mockImplementation(async () => {
      active++;
      maximum = Math.max(maximum, active);
      return beatmap;
    });
    client.assignBeatmap.mockImplementation(async (slotId) => {
      await new Promise<void>((resolve) => release.set(slotId, resolve));
      active--;
      return { id: slotId, code: "", beatmap };
    });
    const run = importPoolBeatmaps({
      client, tournamentId: "cup",
      assignments: ["a", "b", "c", "d"].map((slotId) => ({ slotId, rawInput: "3176982" })),
      onUpdate: () => {},
    });

    await vi.waitFor(() => expect(release.size).toBe(3));
    expect(client.importBeatmapFromUrl).toHaveBeenCalledTimes(3);
    expect(client.getTournament).not.toHaveBeenCalled();
    release.get("a")?.();
    await vi.waitFor(() => expect(release.size).toBe(4));
    expect(client.getTournament).not.toHaveBeenCalled();
    for (const finish of release.values()) finish();
    await run;

    expect(maximum).toBe(3);
    expect(active).toBe(0);
    expect(client.getTournament).toHaveBeenCalledExactlyOnceWith("cup");
  });

  it("surfaces refresh failure without changing a successful Slot outcome", async () => {
    const client = clientFixture();
    client.getTournament.mockRejectedValueOnce(new Error("Refresh failed"));
    const updates: PoolImportUpdate[] = [];

    await expect(importPoolBeatmaps({
      client, tournamentId: "cup", assignments: [{ slotId: "slot", rawInput: "3176982" }],
      onUpdate: (update) => updates.push(update),
    })).rejects.toThrow("Refresh failed");

    expect(updates.at(-1)).toEqual({ slotId: "slot", kind: "assigned" });
  });
});
