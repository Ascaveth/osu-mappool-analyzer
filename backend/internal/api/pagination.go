package api

import (
	"encoding/base64"
	"net/http"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// CursorPage is the pagination envelope every collection endpoint returns
// (docs/api/openapi.yaml CursorPage schema).
type CursorPage struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// parsePageParams reads cursor/limit query params. The cursor is an opaque
// base64-encoded offset — sufficient for an in-memory store; a
// Postgres-backed implementation could switch to a keyset cursor without
// changing this function's signature or callers.
func parsePageParams(r *http.Request) (offset, limit int, ok bool) {
	limit = defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLimit {
			return 0, 0, false
		}
		limit = n
	}

	offset = 0
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil {
			return 0, 0, false
		}
		n, err := strconv.Atoi(string(decoded))
		if err != nil || n < 0 {
			return 0, 0, false
		}
		offset = n
	}

	return offset, limit, true
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// paginate slices items[offset:offset+limit] and builds the CursorPage
// envelope describing what follows.
func paginate[T any](items []T, offset, limit int) ([]T, CursorPage) {
	if offset >= len(items) {
		return []T{}, CursorPage{HasMore: false}
	}
	end := offset + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}
	page := items[offset:end]

	var next *string
	if hasMore {
		c := encodeCursor(end)
		next = &c
	}
	return page, CursorPage{NextCursor: next, HasMore: hasMore}
}
