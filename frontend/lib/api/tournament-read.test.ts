import { expect, it, vi } from "vitest";
import { readTournament, type WireTournament } from "./tournament-read";

it("hydrates duplicate beatmaps once and preserves category codes and difficulty", async () => {
  const beatmap = { id: "b", title: "Map", artist: "Artist", mapper: "Mapper", version: "Hard", ar: 9, od: 8, cs: 4, hp: 5, bpm: 180 };
  const load = vi.fn(async () => new Map([["b", beatmap]]));
  const wire: WireTournament = { id: "t", name: "Open", edition: "", stages: [{ id: "s", name: "Final", order: 1, projected_star_rating: 6, categories: [{ id: "c", name: "HD", order: 1, slots: [1, 2].map(position => ({ id: String(position), position, beatmap_id: "b", effective_difficulty: null })) }] }] };
  const result = await readTournament(wire, load);
  expect(load).toHaveBeenCalledWith(["b"]);
  expect(result.stages[0].categories[0].slots.map(s => s.code)).toEqual(["HD1", "HD2"]);
  expect(result.stages[0].categories[0].slots[0].beatmap).toEqual(beatmap);
  expect(result.stages[0].projectedStarRating).toBe(6);
});

it("propagates hydration failure", async () => {
  await expect(readTournament({ id: "t", name: "Open", edition: "", stages: [] }, async () => { throw new Error("offline"); })).rejects.toThrow("offline");
});
