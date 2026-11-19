package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/sheetimport"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/memory"
)

type importTransport func(*http.Request) (*http.Response, error)

func (transport importTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func TestSheetImportPreviewAndAtomicSave(t *testing.T) {
	s := newTestServer()
	catalog := s.Beatmaps.(*memory.BeatmapRepository)
	pools := s.Tournaments.(*memory.TournamentRepository)
	s.SavePoolImport = func(ctx context.Context, id string, assignments []storage.PoolAssignment) ([]*domain.Beatmap, error) {
		return pools.ImportPool(ctx, catalog, id, assignments)
	}
	ctx := context.Background()
	source, err := os.ReadFile("../osufile/testdata/sample.osu")
	if err != nil {
		t.Fatal(err)
	}
	const sheet = ",Qualifiers,,\n,,MOD#,Map ID\n,,NM1,101\n,,NM2,102\n"
	var sheetStatus, secondMapStatus atomic.Int32
	sheetStatus.Store(200)
	secondMapStatus.Store(200)
	s.sheetImportClient = &http.Client{Transport: importTransport(func(req *http.Request) (*http.Response, error) {
		status := int(sheetStatus.Load())
		body := sheet
		if req.URL.Host == "osu.ppy.sh" {
			id := strings.TrimPrefix(req.URL.Path, "/osu/")
			status = 200
			if id == "102" {
				status = int(secondMapStatus.Load())
			}
			body = strings.Replace(string(source), "BeatmapID:0", "BeatmapID:"+id, 1)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	tournament, err := pools.Save(ctx, &domain.Tournament{Name: "Open", Stages: []domain.Stage{
		{Name: "Qualifiers", Categories: []domain.Category{{Name: "NM", Slots: []domain.Slot{{Position: 1}, {Position: 2}}}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "/v1/tournaments/" + tournament.ID + "/sheet-import"
	router := NewRouter(s)
	post := func(path string, input any, want int) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
		if response.Code != want {
			t.Fatalf("POST %s = %d, want %d: %s", path, response.Code, want, response.Body)
		}
		return response
	}
	preview := func() []sheetimport.Row {
		t.Helper()
		response := post(endpoint+"/preview", map[string]string{"url": "https://docs.google.com/spreadsheets/d/example/edit?gid=0"}, 200)
		var rows []sheetimport.Row
		if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil || len(rows) != 2 {
			t.Fatalf("unexpected preview: %s, %v", response.Body, err)
		}
		return rows
	}
	assertState := func(mapCount int, slotIDs ...int64) {
		t.Helper()
		maps, err := catalog.List(ctx, storage.BeatmapListOptions{})
		if err != nil || len(maps) != mapCount {
			t.Fatalf("catalog has %d maps, want %d: %v", len(maps), mapCount, err)
		}
		current, err := pools.FindByID(ctx, tournament.ID)
		if err != nil {
			t.Fatal(err)
		}
		for i, slot := range current.Stages[0].Categories[0].Slots {
			var got int64
			if slot.Beatmap != nil && slot.Beatmap.OsuBeatmapID != nil {
				got = *slot.Beatmap.OsuBeatmapID
			}
			if got != slotIDs[i] {
				t.Errorf("slot %d = map %d, want %d", i, got, slotIDs[i])
			}
		}
	}

	rows := preview()
	if rows[0].Error != "" || rows[1].Error != "" || rows[0].Checksum == "" || rows[1].Checksum == "" {
		t.Fatalf("preview didn't resolve both maps: %+v", rows)
	}
	assignments := []sheetimport.Assignment{rows[0].Assignment, rows[1].Assignment}
	assertState(0, 0, 0)

	secondMapStatus.Store(404)
	if row := preview()[1]; !strings.Contains(row.Error, "wasn't found") {
		t.Errorf("unexpected missing map message: %q", row.Error)
	}
	post(endpoint, assignments, 422)
	assertState(0, 0, 0)
	secondMapStatus.Store(503)
	if row := preview()[1]; !strings.Contains(row.Error, "Try again") {
		t.Errorf("unexpected temporary error: %q", row.Error)
	}
	secondMapStatus.Store(200)

	conflict := append([]sheetimport.Assignment(nil), assignments...)
	conflict[1].PreviousID = "stale-id"
	post(endpoint, conflict, 409)
	assertState(0, 0, 0)
	// Save uses the reviewed assignments even when the sheet becomes inaccessible.
	sheetStatus.Store(403)
	post(endpoint, assignments, 200)
	assertState(2, 101, 102)
	post(endpoint+"/preview", map[string]string{"url": "https://docs.google.com/spreadsheets/d/example/edit?gid=0"}, 422)
	sheetStatus.Store(200)
	rows = preview()
	if !rows[0].Unchanged || !rows[1].Unchanged {
		t.Fatalf("matching maps should be marked unchanged: %+v", rows)
	}
}
