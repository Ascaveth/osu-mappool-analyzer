package api

import (
	"errors"
	"net/http"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// GetCategory handles GET /categories/{categoryId}.
func (s *Server) GetCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("categoryId")
	cat, _, err := s.Tournaments.FindCategoryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrCategoryNotFound) {
			writeNotFound(w, "category not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toCategoryDTO(r.Context(), *cat, s.StarRatings))
}
