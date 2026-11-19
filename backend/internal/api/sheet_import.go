package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/beatmapimport"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/sheetimport"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

type savePoolImport func(context.Context, string, []storage.PoolAssignment) ([]*domain.Beatmap, error)

var googleExportHost = regexp.MustCompile(`^doc-[a-z0-9-]+-sheets\.googleusercontent\.com$`)

var sheetImportHTTP = &http.Client{
	Timeout: 8 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) < 4 && via[0].URL.Host == "docs.google.com" && req.URL.Scheme == "https" &&
			(req.URL.Host == "docs.google.com" || googleExportHost.MatchString(req.URL.Host)) && req.URL.User == nil {
			return nil
		}
		return http.ErrUseLastResponse
	},
}

func (s *Server) fetchImportSource(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "osu-mappool-analyzer/1.0")
	client := s.sheetImportClient
	if client == nil {
		client = sheetImportHTTP
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, res.StatusCode, nil
	}
	const maxSize = 2 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, maxSize+1))
	if len(body) > maxSize {
		return nil, res.StatusCode, errors.New("import source exceeds 2 MiB")
	}
	return body, res.StatusCode, err
}

func (s *Server) PreviewSheetImport(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input); err != nil {
		writeBadRequest(w, "Paste a Google Sheets tab link and try again.")
		return
	}
	export, err := sheetimport.ExportURL(input.URL)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	tournament, err := s.Tournaments.FindByID(ctx, r.PathValue("tournamentId"))
	if errors.Is(err, storage.ErrTournamentNotFound) {
		writeNotFound(w, "This tournament no longer exists.")
		return
	}
	if err != nil {
		writeProblem(w, 500, "Pool unavailable", "We couldn't load this pool. Try again.")
		return
	}
	data, status, err := s.fetchImportSource(ctx, export)
	if err != nil || status >= 500 || status == http.StatusTooManyRequests {
		writeProblem(w, 502, "Sheet unavailable", "We couldn't read the sheet from Google. Check the link and try again.")
		return
	}
	if status != http.StatusOK {
		writeProblem(w, 422, "Sheet unavailable", "We couldn't open this sheet. Check that anyone with the link can view it, then try again.")
		return
	}
	rows, err := sheetimport.Parse(bytes.NewReader(data))
	if err != nil {
		writeProblem(w, 422, "Unsupported sheet", err.Error())
		return
	}
	sheetimport.MapRows(rows, tournament)
	jobs := make([]sheetimport.Assignment, 0, len(rows))
	for _, row := range rows {
		if row.Error == "" {
			jobs = append(jobs, row.Assignment)
		}
	}
	maps := s.checkSheetMaps(ctx, jobs)
	for i := range rows {
		row := &rows[i]
		if row.Error != "" {
			continue
		}
		result := maps[row.MapID]
		row.Error = result.message
		if result.beatmap != nil {
			mapInfo := result.beatmap
			row.Title = fmt.Sprintf("%s - %s [%s]", mapInfo.Artist, mapInfo.Title, mapInfo.Version)
			row.Checksum = mapInfo.OsuFileHash
		}
	}
	writeJSON(w, 200, rows)
}

type checkedSheetMap struct {
	beatmap *domain.Beatmap
	source  []byte
	message string
}

func (s *Server) checkSheetMaps(ctx context.Context, rows []sheetimport.Assignment) map[int64]checkedSheetMap {
	ids := make(map[int64]bool)
	for _, row := range rows {
		ids[row.MapID] = true
	}
	results := make(map[int64]checkedSheetMap, len(ids))
	var lock sync.Mutex
	var workers sync.WaitGroup
	queue := make(chan int64, len(ids))
	for id := range ids {
		queue <- id
	}
	close(queue)
	for range min(8, len(ids)) {
		workers.Go(func() {
			for id := range queue {
				result := s.checkSheetMap(ctx, id)
				lock.Lock()
				results[id] = result
				lock.Unlock()
			}
		})
	}
	workers.Wait()
	return results
}

func (s *Server) checkSheetMap(ctx context.Context, id int64) checkedSheetMap {
	source, status, err := s.fetchImportSource(ctx, "https://osu.ppy.sh/osu/"+strconv.FormatInt(id, 10))
	switch {
	case status == http.StatusNotFound || status == http.StatusGone:
		return checkedSheetMap{message: fmt.Sprintf("Beatmap %d wasn't found on osu!. Check the Map ID.", id)}
	case err != nil || status != http.StatusOK:
		return checkedSheetMap{message: fmt.Sprintf("We couldn't check beatmap %d on osu!. Try again.", id)}
	}
	beatmap, err := beatmapimport.Prepare(source)
	if err != nil || beatmap.OsuBeatmapID == nil || *beatmap.OsuBeatmapID != id {
		return checkedSheetMap{message: fmt.Sprintf("Beatmap %d didn't return a valid osu! file for that ID. Check the Map ID.", id)}
	}
	return checkedSheetMap{beatmap: beatmap, source: source}
}

func (s *Server) SaveSheetImport(w http.ResponseWriter, r *http.Request) {
	if s.SavePoolImport == nil {
		writeProblem(w, 503, "Import unavailable", "Google Sheets import is unavailable on this server.")
		return
	}
	var rows []sheetimport.Assignment
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10)).Decode(&rows); err != nil || len(rows) == 0 || len(rows) > sheetimport.MaxRows {
		writeBadRequest(w, "Review the sheet again before saving the import.")
		return
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		if row.Row <= 0 || row.SlotID == "" || row.MapID <= 0 || row.MapID > 2147483647 || row.Checksum == "" || seen[row.SlotID] {
			writeBadRequest(w, "The import contains invalid or duplicate assignments. Review the sheet again before saving.")
			return
		}
		seen[row.SlotID] = true
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	maps := s.checkSheetMaps(ctx, rows)
	assignments := make([]storage.PoolAssignment, len(rows))
	for i, row := range rows {
		result := maps[row.MapID]
		if result.message != "" {
			writeProblem(w, 422, "Beatmap unavailable", fmt.Sprintf("Row %d. %s", row.Row, result.message))
			return
		}
		if result.beatmap.OsuFileHash != row.Checksum {
			writeProblem(w, 409, "Beatmap changed", fmt.Sprintf("Beatmap %d changed after you reviewed it. Review the import again before saving.", row.MapID))
			return
		}
		assignments[i] = storage.PoolAssignment{SlotID: row.SlotID, PreviousID: row.PreviousID, Beatmap: result.beatmap}
	}
	saved, err := s.SavePoolImport(ctx, r.PathValue("tournamentId"), assignments)
	switch {
	case errors.Is(err, storage.ErrTournamentNotFound):
		writeNotFound(w, "This tournament no longer exists.")
	case errors.Is(err, storage.ErrPoolImportConflict):
		writeProblem(w, 409, "Pool changed", "A slot changed after you reviewed the import. Review the import again before saving.")
	case err != nil:
		writeProblem(w, 500, "Import failed", "We couldn't save the import. Your pool and beatmap library are unchanged. Try again.")
	default:
		count := 0
		for _, beatmap := range saved {
			if beatmap != nil {
				count++
			}
		}
		writeJSON(w, 200, map[string]int{"saved": count})
		if s.Enricher != nil && count > 0 {
			go s.enrichSheetMaps(saved, maps)
		}
	}
}

func (s *Server) enrichSheetMaps(saved []*domain.Beatmap, maps map[int64]checkedSheetMap) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	seen := make(map[string]bool)
	for _, beatmap := range saved {
		if ctx.Err() != nil {
			return
		}
		if beatmap == nil || beatmap.OsuBeatmapID == nil || seen[beatmap.ID] {
			continue
		}
		seen[beatmap.ID] = true
		if err := s.Enricher.Enrich(ctx, beatmap, maps[*beatmap.OsuBeatmapID].source); err != nil {
			log.Printf("star rating enrichment failed for beatmap %q: %v", beatmap.ID, err)
		}
	}
}
