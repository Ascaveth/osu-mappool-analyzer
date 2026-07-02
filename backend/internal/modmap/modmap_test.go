package modmap

import "testing"

func TestFromCategoryName(t *testing.T) {
	cases := []struct {
		name     string
		wantMods Mods
		wantOK   bool
	}{
		{"NM", NoMod, true},
		{"nm", NoMod, true},
		{"HR", ModHardRock, true},
		{"DT", ModDoubleTime, true},
		{"EZ", ModEasy, true},
		{"HT", ModHalfTime, true},
		{"HD", ModHidden, true},
		{"FL", ModFlashlight, true},
		{"HDHR", ModHidden | ModHardRock, true},
		{"DTHD", ModDoubleTime | ModHidden, true},
		{"FM", 0, false},
		{"TB", 0, false},
		{"", 0, false},
		{"XYZ", 0, false},
		{"HDX", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mods, ok := FromCategoryName(tc.name)
			if ok != tc.wantOK {
				t.Fatalf("FromCategoryName(%q) ok = %v, want %v", tc.name, ok, tc.wantOK)
			}
			if ok && mods != tc.wantMods {
				t.Errorf("FromCategoryName(%q) mods = %v, want %v", tc.name, mods, tc.wantMods)
			}
		})
	}
}

func TestIsFreeMod(t *testing.T) {
	if !IsFreeMod("fm") {
		t.Error("IsFreeMod(\"fm\") = false, want true")
	}
	if IsFreeMod("TB") {
		t.Error("IsFreeMod(\"TB\") = true, want false")
	}
}

func TestFreeModCandidatesExcludesDoubleTime(t *testing.T) {
	for _, m := range FreeModCandidates {
		if m&ModDoubleTime != 0 {
			t.Errorf("FreeModCandidates contains DoubleTime, which is not a legal FreeMod pick: %v", FreeModCandidates)
		}
	}
}

func TestAffectsStarRating(t *testing.T) {
	if AffectsStarRating(ModHidden) {
		t.Error("AffectsStarRating(Hidden) = true, want false")
	}
	if !AffectsStarRating(ModHardRock) {
		t.Error("AffectsStarRating(HardRock) = false, want true")
	}
	if !AffectsStarRating(ModHidden | ModDoubleTime) {
		t.Error("AffectsStarRating(Hidden|DoubleTime) = false, want true")
	}
}
