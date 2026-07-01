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
