export interface CreateCategoryInput {
  order: number;
  modPrefix: string; // e.g. "NM", "HD" — generates slot codes like NM1, NM2; name derived from prefix
  slotCount: number;
}

export interface CreateStageInput {
  name: string;
  order: number;
  categories: CreateCategoryInput[];
}

export interface CreateTournamentInput {
  name: string;
  stages: CreateStageInput[];
}
