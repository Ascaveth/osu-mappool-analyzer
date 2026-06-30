package domain

import "testing"

func validTournament() *Tournament {
	return &Tournament{
		ID: "t-1", Name: "Test Open", Edition: "2026",
		Stages: []Stage{
			{
				ID: "stage-1", Name: "Qualifiers", Order: 1,
				Categories: []Category{
					{ID: "cat-nm", Name: "NM", Order: 1, Slots: []Slot{{ID: "s1", Position: 1}}},
					{ID: "cat-hd", Name: "HD", Order: 2, Slots: []Slot{{ID: "s2", Position: 1}}},
				},
			},
			{
				ID: "stage-2", Name: "Finals", Order: 2,
				Categories: []Category{
					{ID: "cat-finals-nm", Name: "NM", Order: 1, Slots: []Slot{{ID: "s3", Position: 1}}},
				},
			},
		},
	}
}

func TestValidateConfiguration_HappyPathHasNoIssues(t *testing.T) {
	issues := ValidateConfiguration(validTournament())
	if len(issues) != 0 {
		t.Errorf("ValidateConfiguration() = %+v, want no issues", issues)
	}
}

func TestValidateConfiguration_AllowsSameStageOrderAsParallelStages(t *testing.T) {
	tour := validTournament()
	tour.Stages[1].Order = 1 // same order as stage-1, deliberately: a peer-set/parallel format

	issues := ValidateConfiguration(tour)
	if HasErrors(issues) {
		t.Errorf("same-order stages should be a supported parallel-stage format, not an error; got %+v", issues)
	}
}

func TestValidateConfiguration_RejectsDuplicateCategoryOrderWithinStage(t *testing.T) {
	tour := validTournament()
	tour.Stages[0].Categories[1].Order = 1 // collides with NM's order=1

	issues := ValidateConfiguration(tour)
	if !HasErrors(issues) {
		t.Fatal("duplicate category order within a stage should be a hard error")
	}
}

func TestValidateConfiguration_RejectsZeroSlotCategory(t *testing.T) {
	tour := validTournament()
	tour.Stages[0].Categories[0].Slots = nil

	issues := ValidateConfiguration(tour)
	if !HasErrors(issues) {
		t.Fatal("a category with zero slots should be a hard error")
	}
}

func TestValidateConfiguration_WarnsOnDuplicateCategoryNameWithinStage(t *testing.T) {
	tour := validTournament()
	tour.Stages[0].Categories[1].Name = "NM" // now both categories in stage-1 are named "NM"

	issues := ValidateConfiguration(tour)
	if HasErrors(issues) {
		t.Fatalf("duplicate category name should be a warning, not an error; got %+v", issues)
	}
	if len(issues) != 1 {
		t.Fatalf("expected exactly one warning, got %+v", issues)
	}
}

func TestValidateConfiguration_AllowsDuplicateCategoryNamesAcrossDifferentStages(t *testing.T) {
	// validTournament() already has "NM" in both stage-1 and stage-2 — this
	// must not warn, since the rule is scoped per-Stage.
	issues := ValidateConfiguration(validTournament())
	if len(issues) != 0 {
		t.Errorf("category names repeating across different stages should not warn, got %+v", issues)
	}
}

func TestValidateConfiguration_NilTournamentReturnsErrorIssueInsteadOfPanicking(t *testing.T) {
	issues := ValidateConfiguration(nil)
	if !HasErrors(issues) {
		t.Fatalf("ValidateConfiguration(nil) = %+v, want a hard error issue", issues)
	}
}

func TestHasErrors(t *testing.T) {
	if HasErrors(nil) {
		t.Error("HasErrors(nil) = true, want false")
	}
	if HasErrors([]ConfigurationIssue{{IsError: false}}) {
		t.Error("HasErrors with only warnings = true, want false")
	}
	if !HasErrors([]ConfigurationIssue{{IsError: false}, {IsError: true}}) {
		t.Error("HasErrors with a mix should be true")
	}
}
