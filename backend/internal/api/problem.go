// Package api implements the HTTP handlers for docs/api/openapi.yaml — the
// REST surface over the Tournament aggregate, the Beatmap aggregate, and the
// Analysis Engine's output. Handlers translate between domain types and the
// wire (JSON) shapes defined here; they contain no analysis or business
// logic of their own beyond what's needed to call into internal/analysis,
// internal/report, internal/normalize, internal/osufile, and
// internal/storage.
package api

import (
	"encoding/json"
	"net/http"
)

// FieldError is one entry in a Problem's errors[] array (docs/api/openapi.yaml
// Problem schema), used for field-level validation failures.
type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

// Problem is the RFC 7807 application/problem+json body every error
// response in the API uses.
type Problem struct {
	Type   string       `json:"type,omitempty"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}

const problemContentType = "application/problem+json"

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	writeProblemWithErrors(w, status, title, detail, nil)
}

func writeProblemWithErrors(w http.ResponseWriter, status int, title, detail string, fieldErrors []FieldError) {
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{
		Title:  title,
		Status: status,
		Detail: detail,
		Errors: fieldErrors,
	})
}

func writeBadRequest(w http.ResponseWriter, detail string) {
	writeProblem(w, http.StatusBadRequest, "Bad Request", detail)
}

func writeNotFound(w http.ResponseWriter, detail string) {
	writeProblem(w, http.StatusNotFound, "Not Found", detail)
}

func writeValidationError(w http.ResponseWriter, detail string, fieldErrors []FieldError) {
	writeProblemWithErrors(w, http.StatusUnprocessableEntity, "Validation Error", detail, fieldErrors)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
