package memory

import (
	"context"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

func newTournament() *domain.Tournament {
	return &domain.Tournament{
		Name:    "Example Open",
		Edition: "2026",
		Stages: []domain.Stage{
			{
				Name:  "Qualifiers",
				Order: 1,
				Categories: []domain.Category{
					{
						Name:  "NM",
						Order: 1,
						Slots: []domain.Slot{
							{Position: 1},
							{Position: 2},
						},
					},
				},
			},
		},
	}
}

func TestTournamentRepository_SaveAssignsIDsThroughTheTree(t *testing.T) {
	repo := NewTournamentRepository()
	saved, err := repo.Save(context.Background(), newTournament())
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if saved.ID == "" {
		t.Fatal("Save should assign a Tournament ID")
	}
	if saved.Stages[0].ID == "" {
		t.Fatal("Save should assign a Stage ID")
	}
	if saved.Stages[0].Categories[0].ID == "" {
		t.Fatal("Save should assign a Category ID")
	}
	for _, slot := range saved.Stages[0].Categories[0].Slots {
		if slot.ID == "" {
			t.Fatal("Save should assign a Slot ID")
		}
	}
}

func TestTournamentRepository_SaveRejectsNil(t *testing.T) {
	repo := NewTournamentRepository()
	if _, err := repo.Save(context.Background(), nil); err != storage.ErrInvalidTournament {
		t.Errorf("Save(nil) error = %v, want ErrInvalidTournament", err)
	}
}

func TestTournamentRepository_FindByID(t *testing.T) {
	repo := NewTournamentRepository()
	saved, _ := repo.Save(context.Background(), newTournament())

	found, err := repo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if found.Name != "Example Open" {
		t.Errorf("FindByID Name = %q, want %q", found.Name, "Example Open")
	}

	if _, err := repo.FindByID(context.Background(), "missing"); err != storage.ErrTournamentNotFound {
		t.Errorf("FindByID(missing) error = %v, want ErrTournamentNotFound", err)
	}
}

func TestTournamentRepository_ReturnedRecordsAreIsolated(t *testing.T) {
	repo := NewTournamentRepository()
	saved, _ := repo.Save(context.Background(), newTournament())

	saved.Name = "Mutated"
	saved.Stages[0].Categories[0].Slots[0].Position = 99

	found, err := repo.FindByID(context.Background(), saved.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if found.Name != "Example Open" {
		t.Errorf("repository state corrupted: Name = %q", found.Name)
	}
	if found.Stages[0].Categories[0].Slots[0].Position != 1 {
		t.Errorf("repository state corrupted: Slot.Position = %d", found.Stages[0].Categories[0].Slots[0].Position)
	}
}

func TestTournamentRepository_List(t *testing.T) {
	repo := NewTournamentRepository()
	ctx := context.Background()

	a := newTournament()
	a.Name = "Alpha Open"
	repo.Save(ctx, a)

	b := newTournament()
	b.Name = "Beta Cup"
	repo.Save(ctx, b)

	all, err := repo.List(ctx, storage.TournamentListOptions{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List returned %d tournaments, want 2", len(all))
	}
	if all[0].Name != "Alpha Open" || all[1].Name != "Beta Cup" {
		t.Errorf("List order = [%q, %q], want ascending by name", all[0].Name, all[1].Name)
	}

	filtered, err := repo.List(ctx, storage.TournamentListOptions{Query: "beta"})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Name != "Beta Cup" {
		t.Errorf("List(query=beta) = %+v, want just Beta Cup", filtered)
	}

	desc, err := repo.List(ctx, storage.TournamentListOptions{SortDescending: true})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if desc[0].Name != "Beta Cup" {
		t.Errorf("List(desc) order = [%q, %q], want descending by name", desc[0].Name, desc[1].Name)
	}
}

func TestTournamentRepository_Update(t *testing.T) {
	repo := NewTournamentRepository()
	ctx := context.Background()
	saved, _ := repo.Save(ctx, newTournament())

	newName := "Renamed Open"
	updated, err := repo.Update(ctx, saved.ID, storage.TournamentUpdate{Name: &newName})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Name != "Renamed Open" {
		t.Errorf("Update Name = %q, want %q", updated.Name, "Renamed Open")
	}
	if updated.Edition != "2026" {
		t.Errorf("Update should leave Edition unchanged, got %q", updated.Edition)
	}

	if _, err := repo.Update(ctx, "missing", storage.TournamentUpdate{Name: &newName}); err != storage.ErrTournamentNotFound {
		t.Errorf("Update(missing) error = %v, want ErrTournamentNotFound", err)
	}
}

func TestTournamentRepository_Delete(t *testing.T) {
	repo := NewTournamentRepository()
	ctx := context.Background()
	saved, _ := repo.Save(ctx, newTournament())

	if err := repo.Delete(ctx, saved.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := repo.FindByID(ctx, saved.ID); err != storage.ErrTournamentNotFound {
		t.Errorf("FindByID after Delete error = %v, want ErrTournamentNotFound", err)
	}
	if err := repo.Delete(ctx, saved.ID); err != storage.ErrTournamentNotFound {
		t.Errorf("Delete(already deleted) error = %v, want ErrTournamentNotFound", err)
	}
}

func TestTournamentRepository_FindStageByID_FindCategoryByID(t *testing.T) {
	repo := NewTournamentRepository()
	ctx := context.Background()
	saved, _ := repo.Save(ctx, newTournament())

	stageID := saved.Stages[0].ID
	stage, tournamentID, err := repo.FindStageByID(ctx, stageID)
	if err != nil {
		t.Fatalf("FindStageByID returned error: %v", err)
	}
	if stage.ID != stageID || tournamentID != saved.ID {
		t.Errorf("FindStageByID = (%+v, %q), want stage %q under tournament %q", stage, tournamentID, stageID, saved.ID)
	}

	categoryID := saved.Stages[0].Categories[0].ID
	cat, tournamentID2, err := repo.FindCategoryByID(ctx, categoryID)
	if err != nil {
		t.Fatalf("FindCategoryByID returned error: %v", err)
	}
	if cat.ID != categoryID || tournamentID2 != saved.ID {
		t.Errorf("FindCategoryByID = (%+v, %q), want category %q under tournament %q", cat, tournamentID2, categoryID, saved.ID)
	}

	if _, _, err := repo.FindStageByID(ctx, "missing"); err != storage.ErrStageNotFound {
		t.Errorf("FindStageByID(missing) error = %v, want ErrStageNotFound", err)
	}
	if _, _, err := repo.FindCategoryByID(ctx, "missing"); err != storage.ErrCategoryNotFound {
		t.Errorf("FindCategoryByID(missing) error = %v, want ErrCategoryNotFound", err)
	}
}

func TestTournamentRepository_AssignAndClearSlotBeatmap(t *testing.T) {
	repo := NewTournamentRepository()
	ctx := context.Background()
	saved, _ := repo.Save(ctx, newTournament())
	slotID := saved.Stages[0].Categories[0].Slots[0].ID

	bm := &domain.Beatmap{ID: "bm-1", Title: "Test Map"}
	updated, err := repo.AssignSlotBeatmap(ctx, slotID, bm)
	if err != nil {
		t.Fatalf("AssignSlotBeatmap returned error: %v", err)
	}
	if updated.Beatmap == nil || updated.Beatmap.ID != "bm-1" {
		t.Fatalf("AssignSlotBeatmap Slot.Beatmap = %+v, want bm-1", updated.Beatmap)
	}

	found, err := repo.FindByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if found.Stages[0].Categories[0].Slots[0].Beatmap == nil {
		t.Fatal("assignment should be visible through FindByID")
	}

	cleared, err := repo.ClearSlotBeatmap(ctx, slotID)
	if err != nil {
		t.Fatalf("ClearSlotBeatmap returned error: %v", err)
	}
	if cleared.Beatmap != nil {
		t.Errorf("ClearSlotBeatmap Slot.Beatmap = %+v, want nil", cleared.Beatmap)
	}

	if _, err := repo.AssignSlotBeatmap(ctx, "missing", bm); err != storage.ErrSlotNotFound {
		t.Errorf("AssignSlotBeatmap(missing) error = %v, want ErrSlotNotFound", err)
	}
	if _, err := repo.ClearSlotBeatmap(ctx, "missing"); err != storage.ErrSlotNotFound {
		t.Errorf("ClearSlotBeatmap(missing) error = %v, want ErrSlotNotFound", err)
	}
}
