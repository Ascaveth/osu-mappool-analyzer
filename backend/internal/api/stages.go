package api

import (
	"errors"
	"net/http"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// GetStage handles GET /stages/{stageId}.
func (s *Server) GetStage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("stageId")
	stage, _, err := s.Tournaments.FindStageByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrStageNotFound) {
			writeNotFound(w, "stage not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toStageDTO(*stage))
}
