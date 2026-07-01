package storage

import (
	"context"
	"errors"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// ErrTournamentNotFound is returned when a lookup finds no matching tournament.
var ErrTournamentNotFound = errors.New("storage: tournament not found")

// ErrStageNotFound is returned when a Stage ID doesn't resolve to any Stage
// in any stored Tournament.
var ErrStageNotFound = errors.New("storage: stage not found")

// ErrCategoryNotFound is returned when a Category ID doesn't resolve to any
// Category in any stored Tournament.
var ErrCategoryNotFound = errors.New("storage: category not found")

// ErrSlotNotFound is returned when a Slot ID doesn't resolve to any Slot in
// any stored Tournament.
var ErrSlotNotFound = errors.New("storage: slot not found")

// ErrInvalidTournament is returned by Save when given a nil tournament.
var ErrInvalidTournament = errors.New("storage: invalid tournament")

// TournamentUpdate carries the only fields v1's PATCH endpoint may change
// (docs/14-api-specification.md: structural changes require a new
// Tournament, per the aggregate boundary in docs/06-domain-model.md). A nil
// field means "leave unchanged".
type TournamentUpdate struct {
	Name    *string
	Edition *string
}

// TournamentListOptions filters and sorts TournamentRepository.List.
type TournamentListOptions struct {
	// Query is a case-insensitive substring match against Tournament.Name.
	Query string
	// SortDescending reverses the default ascending-by-name order.
	SortDescending bool
}

// TournamentRepository persists and retrieves the Tournament aggregate
// (Tournament -> Stage -> Category -> Slot), edited and read together as one
// unit (docs/06-domain-model.md aggregate boundary). Stage and Category are
// not independently persisted or addressable through this interface except
// via FindStageByID/FindCategoryByID, which resolve a bare ID back to its
// position within the owning Tournament — mirroring the flat
// /stages/{id}, /categories/{id} routes in docs/14-api-specification.md.
type TournamentRepository interface {
	// Save persists a new tournament, assigning IDs to the Tournament and
	// every Stage/Category/Slot within it that doesn't already have one.
	// ID assignment happens here (not in the domain or API layer) so every
	// implementation of this interface — including a future Postgres one —
	// is free to choose its own ID strategy while presenting identical
	// behavior to callers. Returns ErrInvalidTournament for a nil tournament.
	Save(ctx context.Context, t *domain.Tournament) (*domain.Tournament, error)

	// FindByID returns the tournament with the given ID, or ErrTournamentNotFound.
	FindByID(ctx context.Context, id string) (*domain.Tournament, error)

	// List returns tournaments matching opts, in the given sort order.
	List(ctx context.Context, opts TournamentListOptions) ([]domain.Tournament, error)

	// Update applies a partial update (name/edition only) to the tournament
	// with the given ID and returns the updated tournament, or
	// ErrTournamentNotFound.
	Update(ctx context.Context, id string, update TournamentUpdate) (*domain.Tournament, error)

	// Delete removes the tournament with the given ID, or ErrTournamentNotFound.
	Delete(ctx context.Context, id string) error

	// FindStageByID resolves a bare Stage ID to the Stage and the ID of its
	// owning Tournament, or ErrStageNotFound.
	FindStageByID(ctx context.Context, stageID string) (stage *domain.Stage, tournamentID string, err error)

	// FindCategoryByID resolves a bare Category ID to the Category and the
	// ID of its owning Tournament, or ErrCategoryNotFound.
	FindCategoryByID(ctx context.Context, categoryID string) (category *domain.Category, tournamentID string, err error)

	// AssignSlotBeatmap sets the given Slot's Beatmap reference and returns
	// the updated Slot, or ErrSlotNotFound. beatmap must be a fully loaded
	// domain.Beatmap — callers (the API layer) are responsible for fetching
	// it from BeatmapRepository first, since Beatmap is a separate
	// aggregate this repository has no knowledge of.
	AssignSlotBeatmap(ctx context.Context, slotID string, beatmap *domain.Beatmap) (*domain.Slot, error)

	// ClearSlotBeatmap returns the given Slot to its unfilled state and
	// returns the updated Slot, or ErrSlotNotFound.
	ClearSlotBeatmap(ctx context.Context, slotID string) (*domain.Slot, error)
}
