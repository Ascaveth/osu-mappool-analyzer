package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTournament_HappyPath(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(exampleTournamentConfigJSON))
	rec := httptest.NewRecorder()

	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc == "" {
		t.Error("expected Location header on 201")
	}

	var got tournamentDTO
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.ID == "" {
		t.Error("expected non-empty tournament ID")
	}
	if len(got.Stages) != 1 || len(got.Stages[0].Categories) != 2 {
		t.Fatalf("unexpected tree shape: %+v", got)
	}
	if len(got.Stages[0].Categories[0].Slots) != 5 {
		t.Errorf("NM category should have 5 slots, got %d", len(got.Stages[0].Categories[0].Slots))
	}
	for _, slot := range got.Stages[0].Categories[0].Slots {
		if slot.BeatmapID != nil {
			t.Errorf("newly created slot should be unfilled, got beatmap_id = %v", *slot.BeatmapID)
		}
	}
}

func TestCreateTournament_ValidationError(t *testing.T) {
	s := newTestServer()
	body := `{"name": "", "stages": []}`
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(body))
	rec := httptest.NewRecorder()

	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != problemContentType {
		t.Errorf("Content-Type = %q, want %q", ct, problemContentType)
	}
}

func TestCreateTournament_DomainValidationError(t *testing.T) {
	s := newTestServer()
	// Two categories in the same stage sharing order=1 — a hard error in
	// domain.ValidateConfiguration.
	body := `{
		"name": "Bad Open",
		"stages": [{
			"name": "Qualifiers", "order": 1,
			"categories": [
				{ "name": "NM", "order": 1, "slotCount": 1 },
				{ "name": "HD", "order": 1, "slotCount": 1 }
			]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(body))
	rec := httptest.NewRecorder()

	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTournament_MalformedJSON(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(`{not json`))
	rec := httptest.NewRecorder()

	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func createTestTournament(t *testing.T, s *Server) tournamentDTO {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(exampleTournamentConfigJSON))
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: create tournament status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var dto tournamentDTO
	if err := json.NewDecoder(rec.Body).Decode(&dto); err != nil {
		t.Fatalf("setup: decoding tournament: %v", err)
	}
	return dto
}

func TestGetTournament(t *testing.T) {
	s := newTestServer()
	created := createTestTournament(t, s)

	req := httptest.NewRequest(http.MethodGet, "/v1/tournaments/"+created.ID, nil)
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetTournament_NotFound(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/tournaments/missing-id", nil)
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != problemContentType {
		t.Errorf("Content-Type = %q, want %q", ct, problemContentType)
	}
}

func TestUpdateTournament(t *testing.T) {
	s := newTestServer()
	created := createTestTournament(t, s)

	req := httptest.NewRequest(http.MethodPatch, "/v1/tournaments/"+created.ID, strings.NewReader(`{"name": "Renamed"}`))
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got tournamentDTO
	json.NewDecoder(rec.Body).Decode(&got)
	if got.Name != "Renamed" {
		t.Errorf("Name = %q, want %q", got.Name, "Renamed")
	}
}

func TestDeleteTournament(t *testing.T) {
	s := newTestServer()
	created := createTestTournament(t, s)

	req := httptest.NewRequest(http.MethodDelete, "/v1/tournaments/"+created.ID, nil)
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/tournaments/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Errorf("after delete, GET status = %d, want 404", getRec.Code)
	}
}

// --- ProjectedStarRating ---

func TestCreateTournament_ProjectedStarRatingRoundTrips(t *testing.T) {
	s := newTestServer()
	body := `{
		"name": "Example Open",
		"stages": [{
			"name": "Qualifiers", "order": 1,
			"projectedStarRating": 5.5,
			"categories": [{ "name": "NM", "order": 1, "slotCount": 1 }]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(body))
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var got tournamentDTO
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if got.Stages[0].ProjectedStarRating == nil || *got.Stages[0].ProjectedStarRating != 5.5 {
		t.Errorf("ProjectedStarRating = %v, want 5.5", got.Stages[0].ProjectedStarRating)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/tournaments/"+got.ID, nil)
	getRec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(getRec, getReq)
	var refetched tournamentDTO
	if err := json.NewDecoder(getRec.Body).Decode(&refetched); err != nil {
		t.Fatalf("decoding refetch response: %v", err)
	}
	if refetched.Stages[0].ProjectedStarRating == nil || *refetched.Stages[0].ProjectedStarRating != 5.5 {
		t.Errorf("after GET, ProjectedStarRating = %v, want 5.5", refetched.Stages[0].ProjectedStarRating)
	}
}

func TestCreateTournament_ProjectedStarRatingFallsBackToNM1(t *testing.T) {
	s := newTestServer()
	created := createTestTournament(t, s)

	if got := created.Stages[0].ProjectedStarRating; got != nil {
		t.Fatalf("ProjectedStarRating = %v, want nil (no override, NM1 unfilled)", got)
	}

	bm := importTestBeatmap(t, s, "../osufile/testdata/sample.osu")
	nm1SlotID := created.Stages[0].Categories[0].Slots[0].ID
	assignReq := httptest.NewRequest(http.MethodPut, "/v1/slots/"+nm1SlotID+"/beatmap",
		strings.NewReader(`{"beatmap_id": "`+bm.ID+`"}`))
	assignReq.Header.Set("Content-Type", "application/json")
	assignRec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(assignRec, assignReq)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want 200; body = %s", assignRec.Code, assignRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/tournaments/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(getRec, getReq)
	var refetched tournamentDTO
	if err := json.NewDecoder(getRec.Body).Decode(&refetched); err != nil {
		t.Fatalf("decoding refetch response: %v", err)
	}
	if got := refetched.Stages[0].ProjectedStarRating; got == nil || *got != bm.StarRating {
		t.Errorf("ProjectedStarRating = %v, want NM1 beatmap's StarRating (%v)", got, bm.StarRating)
	}
}

func TestCreateTournament_NegativeProjectedStarRatingRejected(t *testing.T) {
	s := newTestServer()
	body := `{
		"name": "Example Open",
		"stages": [{
			"name": "Qualifiers", "order": 1,
			"projectedStarRating": -1,
			"categories": [{ "name": "NM", "order": 1, "slotCount": 1 }]
		}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/tournaments", strings.NewReader(body))
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body = %s", rec.Code, rec.Body.String())
	}
}

func TestListTournaments(t *testing.T) {
	s := newTestServer()
	createTestTournament(t, s)
	createTestTournament(t, s)

	req := httptest.NewRequest(http.MethodGet, "/v1/tournaments", nil)
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got listResponse[tournamentSummaryDTO]
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(got.Data) != 2 {
		t.Fatalf("Data length = %d, want 2", len(got.Data))
	}
}
