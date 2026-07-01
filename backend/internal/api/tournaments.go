package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// ListTournaments handles GET /tournaments.
func (s *Server) ListTournaments(w http.ResponseWriter, r *http.Request) {
	offset, limit, ok := parsePageParams(r)
	if !ok {
		writeBadRequest(w, "invalid cursor or limit")
		return
	}

	opts := storage.TournamentListOptions{
		Query:          r.URL.Query().Get("q"),
		SortDescending: r.URL.Query().Get("sort") == "-name",
	}

	all, err := s.Tournaments.List(r.Context(), opts)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	page, cursorPage := paginate(all, offset, limit)
	data := make([]tournamentSummaryDTO, len(page))
	for i, t := range page {
		data[i] = toTournamentSummaryDTO(t)
	}

	writeJSON(w, http.StatusOK, listResponse[tournamentSummaryDTO]{Data: data, Pagination: cursorPage})
}

type listResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination CursorPage `json:"pagination"`
}

// CreateTournament handles POST /tournaments.
func (s *Server) CreateTournament(w http.ResponseWriter, r *http.Request) {
	var dto tournamentConfigurationDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeBadRequest(w, "invalid JSON body: "+err.Error())
		return
	}

	if fieldErrs := validateTournamentConfiguration(dto); len(fieldErrs) > 0 {
		writeValidationError(w, "tournament configuration failed validation", fieldErrs)
		return
	}

	tournament := dto.toDomain()
	if issues := domain.ValidateConfiguration(tournament); domain.HasErrors(issues) {
		writeValidationError(w, configurationIssuesDetail(issues), nil)
		return
	}

	saved, err := s.Tournaments.Save(r.Context(), tournament)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	w.Header().Set("Location", "/v1/tournaments/"+saved.ID)
	writeJSON(w, http.StatusCreated, toTournamentDTO(saved))
}

func configurationIssuesDetail(issues []domain.ConfigurationIssue) string {
	for _, issue := range issues {
		if issue.IsError {
			return issue.Message
		}
	}
	return "invalid tournament configuration"
}

// GetTournament handles GET /tournaments/{tournamentId}.
func (s *Server) GetTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("tournamentId")
	t, err := s.Tournaments.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrTournamentNotFound) {
			writeNotFound(w, "tournament not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTournamentDTO(t))
}

// UpdateTournament handles PATCH /tournaments/{tournamentId}.
func (s *Server) UpdateTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("tournamentId")

	var dto tournamentUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeBadRequest(w, "invalid JSON body: "+err.Error())
		return
	}
	if dto.Name == nil && dto.Edition == nil {
		writeBadRequest(w, "at least one of name or edition must be provided")
		return
	}
	if dto.Name != nil && *dto.Name == "" {
		writeBadRequest(w, "name must not be empty")
		return
	}

	updated, err := s.Tournaments.Update(r.Context(), id, storage.TournamentUpdate{Name: dto.Name, Edition: dto.Edition})
	if err != nil {
		if errors.Is(err, storage.ErrTournamentNotFound) {
			writeNotFound(w, "tournament not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTournamentDTO(updated))
}

// DeleteTournament handles DELETE /tournaments/{tournamentId}.
func (s *Server) DeleteTournament(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("tournamentId")
	if err := s.Tournaments.Delete(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrTournamentNotFound) {
			writeNotFound(w, "tournament not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
