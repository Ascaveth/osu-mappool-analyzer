package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

type assignSlotBeatmapDTO struct {
	BeatmapID string `json:"beatmap_id"`
}

// AssignSlotBeatmap handles PUT /slots/{slotId}/beatmap.
func (s *Server) AssignSlotBeatmap(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")

	var dto assignSlotBeatmapDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeBadRequest(w, "invalid JSON body: "+err.Error())
		return
	}
	if dto.BeatmapID == "" {
		writeBadRequest(w, "beatmap_id must not be empty")
		return
	}

	beatmap, err := s.Beatmaps.FindByID(r.Context(), dto.BeatmapID)
	if err != nil {
		if errors.Is(err, storage.ErrBeatmapNotFound) {
			writeNotFound(w, "beatmap not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	slot, err := s.Tournaments.AssignSlotBeatmap(r.Context(), slotID, beatmap)
	if err != nil {
		if errors.Is(err, storage.ErrSlotNotFound) {
			writeNotFound(w, "slot not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toSlotDTO(*slot))
}

// UnassignSlotBeatmap handles DELETE /slots/{slotId}/beatmap.
func (s *Server) UnassignSlotBeatmap(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slotId")

	if _, err := s.Tournaments.ClearSlotBeatmap(r.Context(), slotID); err != nil {
		if errors.Is(err, storage.ErrSlotNotFound) {
			writeNotFound(w, "slot not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
