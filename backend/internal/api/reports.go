package api

import (
	"errors"
	"net/http"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/report"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// GetTournamentReport handles GET /tournaments/{tournamentId}/report.
func (s *Server) GetTournamentReport(w http.ResponseWriter, r *http.Request) {
	tournamentID := r.PathValue("tournamentId")

	tournament, analyses, err := s.runEngine(r, tournamentID)
	if err != nil {
		if errors.Is(err, storage.ErrTournamentNotFound) {
			writeNotFound(w, "tournament not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	q := r.URL.Query()
	scopeType := q.Get("scope_type")
	scopeID := q.Get("scope_id")

	scope := domain.Scope{Type: domain.ScopeTournament, ID: tournament.ID}
	if scopeType != "" && scopeID != "" {
		scope = domain.Scope{Type: domain.ScopeType(scopeType), ID: scopeID}
		filtered := make([]domain.Analysis, 0, len(analyses))
		for _, a := range analyses {
			if a.Scope.Type == scope.Type && a.Scope.ID == scope.ID {
				filtered = append(filtered, a)
			}
		}
		analyses = filtered
	}

	rep := report.Build(scope, analyses, s.Now)
	writeJSON(w, http.StatusOK, toReportDTO(rep))
}
