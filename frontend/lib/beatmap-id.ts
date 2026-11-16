/**
 * Extract a beatmap difficulty ID from a pasted URL or bare numeric ID.
 *
 * Accepts:
 * - https://osu.ppy.sh/beatmapsets/{set}#{mode}/{id}
 * - https://osu.ppy.sh/beatmaps/{id}
 * - bare numeric ID
 */
export function extractBeatmapId(url: string): string {
  const trimmed = url.trim();
  const setHash = trimmed.match(/beatmapsets\/\d+#[a-z]+\/(\d+)/);
  if (setHash) return setHash[1];
  const beatmapPath = trimmed.match(/beatmaps\/(\d+)/);
  if (beatmapPath) return beatmapPath[1];
  const bare = trimmed.match(/^(\d+)$/);
  if (bare) return bare[1];
  throw new Error(`Cannot parse beatmap ID from URL: ${url}`);
}
