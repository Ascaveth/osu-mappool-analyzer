export interface CreateCategoryInput {
  order: number;
  modPrefix: string; // e.g. "NM", "HD" — generates slot codes like NM1, NM2; name derived from prefix
  slotCount: number;
}

export interface CreateStageInput {
  name: string;
  order: number;
  categories: CreateCategoryInput[];
  // Optional organizer-entered target Star Rating; omitted means unset
  // (falls back to the stage's NM1 beatmap's star rating once filled).
  projectedStarRating?: number;
}

export interface CreateTournamentInput {
  name: string;
  stages: CreateStageInput[];
}
