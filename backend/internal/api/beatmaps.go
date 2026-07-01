package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/normalize"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osufile"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

var beatmapSortKeys = map[string]bool{
	"title": true, "-title": true,
	"bpm": true, "-bpm": true,
	"length_seconds": true, "-length_seconds": true,
}

// ListBeatmaps handles GET /beatmaps.
func (s *Server) ListBeatmaps(w http.ResponseWriter, r *http.Request) {
	offset, limit, ok := parsePageParams(r)
	if !ok {
		writeBadRequest(w, "invalid cursor or limit")
		return
	}

	q := r.URL.Query()
	opts := storage.BeatmapListOptions{
		Query:  q.Get("q"),
		Mapper: q.Get("mapper"),
	}

	if raw := q.Get("bpm_min"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeBadRequest(w, "invalid bpm_min")
			return
		}
		opts.BPMMin = &v
	}
	if raw := q.Get("bpm_max"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeBadRequest(w, "invalid bpm_max")
			return
		}
		opts.BPMMax = &v
	}

	sortParam := q.Get("sort")
	if sortParam == "" {
		sortParam = "title"
	}
	if !beatmapSortKeys[sortParam] {
		writeBadRequest(w, "invalid sort key")
		return
	}
	opts.SortDescending = len(sortParam) > 0 && sortParam[0] == '-'
	opts.SortBy = sortParam
	if opts.SortDescending {
		opts.SortBy = sortParam[1:]
	}

	all, err := s.Beatmaps.List(r.Context(), opts)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	page, cursorPage := paginate(all, offset, limit)
	data := make([]beatmapDTO, len(page))
	for i, b := range page {
		data[i] = toBeatmapDTO(&b)
	}
	writeJSON(w, http.StatusOK, listResponse[beatmapDTO]{Data: data, Pagination: cursorPage})
}

// ImportBeatmap handles POST /beatmaps.
func (s *Server) ImportBeatmap(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		writeBadRequest(w, "missing multipart field \"file\": "+err.Error())
		return
	}
	defer file.Close()

	source, err := io.ReadAll(file)
	if err != nil {
		writeBadRequest(w, "failed to read uploaded file: "+err.Error())
		return
	}

	raw, err := osufile.Parse(bytes.NewReader(source))
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "Unparseable Beatmap", "file is not a parseable .osu file: "+err.Error())
		return
	}
	beatmap, err := normalize.Beatmap(raw, source)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "Unparseable Beatmap", "failed to normalize beatmap: "+err.Error())
		return
	}

	_, err = s.Beatmaps.FindByHash(r.Context(), beatmap.OsuFileHash)
	isNew := errors.Is(err, storage.ErrBeatmapNotFound)
	if err != nil && !isNew {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	saved, err := s.Beatmaps.Save(r.Context(), beatmap)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	if isNew {
		w.Header().Set("Location", "/v1/beatmaps/"+saved.ID)
		writeJSON(w, http.StatusCreated, toBeatmapDTO(saved))
		return
	}
	writeJSON(w, http.StatusOK, toBeatmapDTO(saved))
}

// GetBeatmap handles GET /beatmaps/{beatmapId}.
func (s *Server) GetBeatmap(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("beatmapId")
	b, err := s.Beatmaps.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrBeatmapNotFound) {
			writeNotFound(w, "beatmap not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toBeatmapDTO(b))
}
