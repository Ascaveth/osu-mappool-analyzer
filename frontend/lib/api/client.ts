import type { Tournament, Beatmap, Slot, Report } from "@/lib/types";
import type { CreateTournamentInput } from "./types";

export interface ApiClient {
  createTournament(input: CreateTournamentInput): Promise<Tournament>;
  getTournament(id: string): Promise<Tournament>;
  importBeatmapFromUrl(url: string): Promise<Beatmap>;
  listBeatmaps(): Promise<Beatmap[]>;
  assignBeatmap(slotId: string, beatmapId: string): Promise<Slot>;
  clearBeatmap(slotId: string): Promise<Slot>;
  getReport(tournamentId: string): Promise<Report>;
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly detail: string,
    public readonly type?: string,
  ) {
    super(detail);
    this.name = "ApiError";
  }
}
