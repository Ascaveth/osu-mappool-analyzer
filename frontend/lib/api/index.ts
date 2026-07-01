import { createMockClient } from "./mock";
import { createRestClient } from "./rest";

export { createMockClient } from "./mock";
export { createRestClient } from "./rest";
export type { ApiClient } from "./client";
export { ApiError } from "./client";
export type { CreateTournamentInput, CreateStageInput, CreateCategoryInput } from "./types";

const baseUrl = process.env.NEXT_PUBLIC_API_BASE_URL;

// With no backend configured, fall back to the localStorage-backed mock so
// the app still demos standalone (see mock.ts's synthesized "demo mode"
// finding). Once NEXT_PUBLIC_API_BASE_URL points at a running
// cmd/server, the real client takes over.
export const api = baseUrl ? createRestClient(baseUrl) : createMockClient();
