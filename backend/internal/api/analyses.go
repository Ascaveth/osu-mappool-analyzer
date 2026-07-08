package api

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// runEngine loads the tournament and runs every registered analyzer against
// it. Per-analyzer errors are logged but don't fail the request — a defect
// in one analyzer must not hide the results of the others (same contract
// analysis.Engine.Run documents).
func (s *Server) runEngine(r *http.Request, tournamentID string) (*domain.Tournament, []domain.Analysis, error) {
	tournament, err := s.Tournaments.FindByID(r.Context(), tournamentID)
	if err != nil {
		return nil, nil, err
	}
	analyses, runErr := s.Engine.Run(r.Context(), tournament)
	if runErr != nil {
		log.Printf("engine run for tournament %q had errors: %v", tournamentID, runErr)
	}
	return tournament, analyses, nil
}

// ListTournamentAnalyses handles GET /tournaments/{tournamentId}/analyses.
func (s *Server) ListTournamentAnalyses(w http.ResponseWriter, r *http.Request) {
	tournamentID := r.PathValue("tournamentId")

	offset, limit, ok := parsePageParams(r)
	if !ok {
		writeBadRequest(w, "invalid cursor or limit")
		return
	}

	_, analyses, err := s.runEngine(r, tournamentID)
	if err != nil {
		if errors.Is(err, storage.ErrTournamentNotFound) {
			writeNotFound(w, "tournament not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	q := r.URL.Query()
	analyzerName := q.Get("analyzer_name")
	scopeType := q.Get("scope_type")
	scopeID := q.Get("scope_id")
	var severities map[domain.Severity]bool
	if raw := q.Get("severity"); raw != "" {
		severities = map[domain.Severity]bool{}
		for _, part := range strings.Split(raw, ",") {
			severities[domain.Severity(strings.TrimSpace(part))] = true
		}
	}

	filtered := make([]domain.Analysis, 0, len(analyses))
	for _, a := range analyses {
		if analyzerName != "" && a.AnalyzerName != analyzerName {
			continue
		}
		if scopeType != "" && string(a.Scope.Type) != scopeType {
			continue
		}
		if scopeID != "" && a.Scope.ID != scopeID {
			continue
		}
		if severities != nil {
			a.Findings = filterFindingsBySeverity(a.Findings, severities)
		}
		filtered = append(filtered, a)
	}

	page, cursorPage := paginate(filtered, offset, limit)
	data := make([]analysisDTO, len(page))
	for i, a := range page {
		data[i] = toAnalysisDTO(a)
	}
	writeJSON(w, http.StatusOK, listResponse[analysisDTO]{Data: data, Pagination: cursorPage})
}

func filterFindingsBySeverity(findings []domain.Finding, severities map[domain.Severity]bool) []domain.Finding {
	out := make([]domain.Finding, 0, len(findings))
	for _, f := range findings {
		if severities[f.Severity] {
			out = append(out, f)
		}
	}
	return out
}
