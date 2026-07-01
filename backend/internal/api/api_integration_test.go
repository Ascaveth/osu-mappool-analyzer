package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFullFlow_CreateImportAssignAnalyzeReport exercises the exact path the
// frontend uses end to end: create a tournament, import a real .osu
// fixture, assign it to a slot, list analyses, and fetch a report — the
// HTTP-level equivalent of internal/integration/pipeline_test.go.
func TestFullFlow_CreateImportAssignAnalyzeReport(t *testing.T) {
	s := newTestServer()
	router := NewRouter(s)

	tournament := createTestTournament(t, s)
	slotID := tournament.Stages[0].Categories[0].Slots[0].ID

	beatmap := importTestBeatmap(t, s, "../osufile/testdata/sample.osu")

	assignBody := `{"beatmap_id": "` + beatmap.ID + `"}`
	assignReq := httptest.NewRequest(http.MethodPut, "/v1/slots/"+slotID+"/beatmap", strings.NewReader(assignBody))
	assignRec := httptest.NewRecorder()
	router.ServeHTTP(assignRec, assignReq)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want 200; body = %s", assignRec.Code, assignRec.Body.String())
	}
	var slot slotDTO
	json.NewDecoder(assignRec.Body).Decode(&slot)
	if slot.BeatmapID == nil || *slot.BeatmapID != beatmap.ID {
		t.Fatalf("assigned slot beatmap_id = %v, want %q", slot.BeatmapID, beatmap.ID)
	}

	analysesReq := httptest.NewRequest(http.MethodGet, "/v1/tournaments/"+tournament.ID+"/analyses", nil)
	analysesRec := httptest.NewRecorder()
	router.ServeHTTP(analysesRec, analysesReq)
	if analysesRec.Code != http.StatusOK {
		t.Fatalf("analyses status = %d, want 200; body = %s", analysesRec.Code, analysesRec.Body.String())
	}
	var analyses listResponse[analysisDTO]
	json.NewDecoder(analysesRec.Body).Decode(&analyses)
	if len(analyses.Data) == 0 {
		t.Fatal("expected at least one Analysis")
	}

	reportReq := httptest.NewRequest(http.MethodGet, "/v1/tournaments/"+tournament.ID+"/report", nil)
	reportRec := httptest.NewRecorder()
	router.ServeHTTP(reportRec, reportReq)
	if reportRec.Code != http.StatusOK {
		t.Fatalf("report status = %d, want 200; body = %s", reportRec.Code, reportRec.Body.String())
	}
	var rep reportDTO
	json.NewDecoder(reportRec.Body).Decode(&rep)
	if rep.Sections.Summary == "" {
		t.Error("Report.Sections.Summary should not be empty")
	}

	unassignReq := httptest.NewRequest(http.MethodDelete, "/v1/slots/"+slotID+"/beatmap", nil)
	unassignRec := httptest.NewRecorder()
	router.ServeHTTP(unassignRec, unassignReq)
	if unassignRec.Code != http.StatusNoContent {
		t.Fatalf("unassign status = %d, want 204; body = %s", unassignRec.Code, unassignRec.Body.String())
	}
}

func TestAssignSlotBeatmap_UnknownBeatmap(t *testing.T) {
	s := newTestServer()
	router := NewRouter(s)
	tournament := createTestTournament(t, s)
	slotID := tournament.Stages[0].Categories[0].Slots[0].ID

	req := httptest.NewRequest(http.MethodPut, "/v1/slots/"+slotID+"/beatmap", strings.NewReader(`{"beatmap_id": "missing"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}

func TestAssignSlotBeatmap_UnknownSlot(t *testing.T) {
	s := newTestServer()
	router := NewRouter(s)
	beatmap := importTestBeatmap(t, s, "../osufile/testdata/sample.osu")

	req := httptest.NewRequest(http.MethodPut, "/v1/slots/missing/beatmap", strings.NewReader(`{"beatmap_id": "`+beatmap.ID+`"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetStageAndCategory(t *testing.T) {
	s := newTestServer()
	router := NewRouter(s)
	tournament := createTestTournament(t, s)

	stageReq := httptest.NewRequest(http.MethodGet, "/v1/stages/"+tournament.Stages[0].ID, nil)
	stageRec := httptest.NewRecorder()
	router.ServeHTTP(stageRec, stageReq)
	if stageRec.Code != http.StatusOK {
		t.Fatalf("GetStage status = %d, want 200; body = %s", stageRec.Code, stageRec.Body.String())
	}

	catReq := httptest.NewRequest(http.MethodGet, "/v1/categories/"+tournament.Stages[0].Categories[0].ID, nil)
	catRec := httptest.NewRecorder()
	router.ServeHTTP(catRec, catReq)
	if catRec.Code != http.StatusOK {
		t.Fatalf("GetCategory status = %d, want 200; body = %s", catRec.Code, catRec.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/v1/stages/missing", nil)
	missingRec := httptest.NewRecorder()
	router.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("GetStage(missing) status = %d, want 404", missingRec.Code)
	}
}
