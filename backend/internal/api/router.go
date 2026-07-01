package api

import "net/http"

// NewRouter builds the *http.ServeMux serving every operation in
// docs/api/openapi.yaml, mounted under /v1 per the spec's URI-based
// versioning policy. Uses Go 1.22+'s method+wildcard ServeMux patterns —
// no third-party router needed for this route count.
func NewRouter(s *Server) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/tournaments", s.ListTournaments)
	mux.HandleFunc("POST /v1/tournaments", s.CreateTournament)
	mux.HandleFunc("GET /v1/tournaments/{tournamentId}", s.GetTournament)
	mux.HandleFunc("PATCH /v1/tournaments/{tournamentId}", s.UpdateTournament)
	mux.HandleFunc("DELETE /v1/tournaments/{tournamentId}", s.DeleteTournament)

	mux.HandleFunc("GET /v1/stages/{stageId}", s.GetStage)
	mux.HandleFunc("GET /v1/categories/{categoryId}", s.GetCategory)

	mux.HandleFunc("PUT /v1/slots/{slotId}/beatmap", s.AssignSlotBeatmap)
	mux.HandleFunc("DELETE /v1/slots/{slotId}/beatmap", s.UnassignSlotBeatmap)

	mux.HandleFunc("GET /v1/beatmaps", s.ListBeatmaps)
	mux.HandleFunc("POST /v1/beatmaps", s.ImportBeatmap)
	mux.HandleFunc("GET /v1/beatmaps/{beatmapId}", s.GetBeatmap)

	mux.HandleFunc("GET /v1/tournaments/{tournamentId}/analyses", s.ListTournamentAnalyses)
	mux.HandleFunc("GET /v1/tournaments/{tournamentId}/report", s.GetTournamentReport)

	return mux
}
