import type { ApiClient } from "@/lib/api/client";
import type { BulkPasteAssignment } from "@/lib/bulk-paste";

export type PoolImportAssignment = Pick<BulkPasteAssignment, "slotId" | "rawInput">;

export type PoolImportUpdate = { slotId: string } & (
  | { kind: "importing" }
  | { kind: "assigned" }
  | { kind: "failed"; message: string }
);

export async function importPoolBeatmaps({
  client,
  tournamentId,
  assignments,
  onUpdate,
}: {
  client: Pick<ApiClient, "importBeatmapFromUrl" | "assignBeatmap" | "getTournament">;
  tournamentId: string;
  assignments: PoolImportAssignment[];
  onUpdate: (update: PoolImportUpdate) => void;
}) {
  const queue = [...assignments];
  const worker = async () => {
    let assignment: PoolImportAssignment | undefined;
    while ((assignment = queue.shift()) !== undefined) {
      const { slotId, rawInput } = assignment;
      onUpdate({ slotId, kind: "importing" });
      let outcome: PoolImportUpdate;
      try {
        const beatmap = await client.importBeatmapFromUrl(rawInput.trim());
        await client.assignBeatmap(slotId, beatmap.id);
        outcome = { slotId, kind: "assigned" };
      } catch (error) {
        outcome = {
          slotId,
          kind: "failed",
          message: `${error instanceof Error ? error.message : "Import failed"}. Use a beatmap ID such as 3176982 or a full beatmap URL, check your connection, and import again.`,
        };
      }
      onUpdate(outcome);
    }
  };
  await Promise.all(Array.from({ length: Math.min(3, queue.length) }, worker));
  return client.getTournament(tournamentId);
}
