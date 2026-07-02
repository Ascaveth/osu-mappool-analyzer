package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// failingEnricher always returns an error, so tests can assert that
// enrichment failure never fails the import HTTP response.
type failingEnricher struct{ called bool }

func (f *failingEnricher) Enrich(_ context.Context, _ *domain.Beatmap, _ []byte) error {
	f.called = true
	return errors.New("osu! API unreachable")
}

func importTestBeatmap(t *testing.T, s *Server, path string) beatmapDTO {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", path, err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "sample.osu")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	fw.Write(source)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/beatmaps", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("import status = %d, want 200 or 201; body = %s", rec.Code, rec.Body.String())
	}
	var dto beatmapDTO
	if err := json.NewDecoder(rec.Body).Decode(&dto); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return dto
}

func TestImportBeatmap_NewFile(t *testing.T) {
	s := newTestServer()
	dto := importTestBeatmap(t, s, "../osufile/testdata/sample.osu")

	if dto.ID == "" {
		t.Error("expected non-empty beatmap ID")
	}
	if dto.OsuFileHash == "" {
		t.Error("expected non-empty osu_file_hash")
	}
}

func TestImportBeatmap_DeduplicatesByHash(t *testing.T) {
	s := newTestServer()
	first := importTestBeatmap(t, s, "../osufile/testdata/sample.osu")

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	source, _ := os.ReadFile("../osufile/testdata/sample.osu")
	fw, _ := w.CreateFormFile("file", "sample.osu")
	fw.Write(source)
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/beatmaps", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("re-import status = %d, want 200 (deduplicated); body = %s", rec.Code, rec.Body.String())
	}
	var second beatmapDTO
	json.NewDecoder(rec.Body).Decode(&second)
	if second.ID != first.ID {
		t.Errorf("re-import ID = %q, want existing ID %q", second.ID, first.ID)
	}
}

func TestImportBeatmap_MissingFile(t *testing.T) {
	s := newTestServer()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/beatmaps", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestImportBeatmap_EnrichmentFailureDoesNotFailImport(t *testing.T) {
	s := newTestServer()
	enricher := &failingEnricher{}
	s.Enricher = enricher

	dto := importTestBeatmap(t, s, "../osufile/testdata/sample.osu")

	if dto.ID == "" {
		t.Error("expected import to succeed despite enrichment failure")
	}
	if !enricher.called {
		t.Error("expected Enricher.Enrich to be called")
	}
}

func TestGetBeatmap_NotFound(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/beatmaps/missing", nil)
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}

func TestListBeatmaps(t *testing.T) {
	s := newTestServer()
	importTestBeatmap(t, s, "../osufile/testdata/sample.osu")

	req := httptest.NewRequest(http.MethodGet, "/v1/beatmaps", nil)
	rec := httptest.NewRecorder()
	NewRouter(s).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got listResponse[beatmapDTO]
	json.NewDecoder(rec.Body).Decode(&got)
	if len(got.Data) != 1 {
		t.Fatalf("Data length = %d, want 1", len(got.Data))
	}
}
