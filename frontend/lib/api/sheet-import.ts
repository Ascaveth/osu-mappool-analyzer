export interface SheetAssignment {
  row: number;
  slot_id: string;
  map_id: number;
  previous_id: string;
  checksum: string;
}

export interface SheetImportRow extends SheetAssignment {
  stage: string;
  slot: string;
  title: string;
  previous?: string;
  unchanged: boolean;
  error?: string;
  warning?: string;
}

export interface SheetImportClient {
  review(tournamentId: string, url: string): Promise<SheetImportRow[]>;
  save(tournamentId: string, assignments: SheetAssignment[]): Promise<{ saved: number }>;
}
