package pattern

import (
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func TestComputeSkillsetProfile_WideJumps(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		circle(0, 0, 0), circle(300, 0, 300), circle(0, 0, 600),
	}}

	profile := ComputeSkillsetProfile(bm)
	if profile.AvgJumpDistance != 300 {
		t.Errorf("AvgJumpDistance = %v, want 300", profile.AvgJumpDistance)
	}
}

func TestComputeSkillsetProfile_Stream(t *testing.T) {
	var objects []domain.HitObject
	for i := 0; i < 8; i++ {
		objects = append(objects, circle(0, 0, i*80))
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	profile := ComputeSkillsetProfile(bm)
	if profile.StreamCount != 1 {
		t.Errorf("StreamCount = %v, want 1", profile.StreamCount)
	}
	if profile.LongestRunLength != 8 {
		t.Errorf("LongestRunLength = %v, want 8", profile.LongestRunLength)
	}
}

func TestComputeSkillsetProfile_RegularJumpstreamHasZeroSpacingCV(t *testing.T) {
	var objects []domain.HitObject
	for i := 0; i < 8; i++ {
		x := 0
		if i%2 == 1 {
			x = 300
		}
		objects = append(objects, domain.HitObject{Type: domain.HitObjectCircle, X: x, StartTime: ms(i * 80)})
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	profile := ComputeSkillsetProfile(bm)
	if profile.StreamCount != 1 {
		t.Fatalf("StreamCount = %v, want 1", profile.StreamCount)
	}
	if profile.MaxStreamSpacingCV != 0 {
		t.Errorf("MaxStreamSpacingCV = %v, want 0 (perfectly even spacing)", profile.MaxStreamSpacingCV)
	}
}

func TestComputeSkillsetProfile_IrregularJumpstreamHasHighSpacingCV(t *testing.T) {
	x := 0
	var objects []domain.HitObject
	for i := 0; i < 8; i++ {
		objects = append(objects, domain.HitObject{Type: domain.HitObjectCircle, X: x, StartTime: ms(i * 80)})
		if i%2 == 0 {
			x += 300
		} else {
			x += 100
		}
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	profile := ComputeSkillsetProfile(bm)
	if profile.MaxStreamSpacingCV < 0.35 {
		t.Errorf("MaxStreamSpacingCV = %v, want >= 0.35 (spacing alternates 300px/100px)", profile.MaxStreamSpacingCV)
	}
}

func TestComputeSkillsetProfile_CarriesBPM(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", BPM: 120, HitObjects: []domain.HitObject{circle(0, 0, 0)}}

	profile := ComputeSkillsetProfile(bm)
	if profile.BPM != 120 {
		t.Errorf("BPM = %v, want 120", profile.BPM)
	}
}

func TestComputeSkillsetProfile_SliderComplexity(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		slider(0, 0, 0, 5, 1),
		slider(0, 0, 1000, 3, 0),
	}}

	profile := ComputeSkillsetProfile(bm)
	if profile.AvgAnchorCount != 4 {
		t.Errorf("AvgAnchorCount = %v, want 4", profile.AvgAnchorCount)
	}
	if profile.ReverseSliderRatio != 0.5 {
		t.Errorf("ReverseSliderRatio = %v, want 0.5", profile.ReverseSliderRatio)
	}
}

func TestComputeSkillsetProfile_ZeroBPMDoesNotPanic(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", BPM: 0, HitObjects: []domain.HitObject{
		circle(0, 0, 0), circle(0, 0, 80), circle(0, 0, 160),
	}}

	profile := ComputeSkillsetProfile(bm)
	if profile.StreamCount != 0 {
		t.Errorf("StreamCount = %v, want 0 (zero BPM must not classify as stream)", profile.StreamCount)
	}
}

func TestComputeSkillsetProfile_EmptyHitObjectsDoesNotPanic(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1"}

	profile := ComputeSkillsetProfile(bm)
	if profile.ObjectCount != 0 {
		t.Errorf("ObjectCount = %v, want 0", profile.ObjectCount)
	}
}
