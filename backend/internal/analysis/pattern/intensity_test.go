package pattern

import (
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"testing"
	"time"
)

func TestIntensityTimingAndMods(t *testing.T) {
	bm := &domain.Beatmap{BPM: 120, CS: 4}
	for i := 0; i < 16; i++ {
		bm.HitObjects = append(bm.HitObjects, domain.HitObject{StartTime: time.Duration(i) * 125 * time.Millisecond, X: i * 20})
	}
	nm, _ := ComputePlacementSkillsetProfile(bm, "NM")
	dt, _ := ComputePlacementSkillsetProfile(bm, "DT")
	if nm.TappingRate != 8 || dt.TappingRate != 12 || dt.AimSpeed <= nm.AimSpeed || nm.StreamNoteRatio != 1 {
		t.Fatalf("NM %+v DT %+v", nm, dt)
	}
	bm.HitObjects[15].StartTime += 125 * time.Millisecond
	if ComputeSkillsetProfile(bm).TappingRate >= 8 {
		t.Fatal("local timing ignored")
	}
	if ComputeSkillsetProfile(nil).TappingRate != 0 {
		t.Fatal("nil input")
	}
	bm.HitObjects = bm.HitObjects[:1]
	if ComputeSkillsetProfile(bm).TappingRate != 0 {
		t.Fatal("single object")
	}
}

func TestIntensityDistributionAndLength(t *testing.T) {
	makeMap := func(gap time.Duration, count int) *domain.Beatmap {
		bm := &domain.Beatmap{BPM: 120}
		for block := 0; block < count; block++ {
			for i := 0; i < 8; i++ {
				bm.HitObjects = append(bm.HitObjects, domain.HitObject{StartTime: time.Duration(block)*gap + time.Duration(i)*125*time.Millisecond})
			}
		}
		return bm
	}
	short := ComputeSkillsetProfile(makeMap(20*time.Second, 1))
	long := ComputeSkillsetProfile(makeMap(20*time.Second, 2))
	concentrated := ComputeSkillsetProfile(makeMap(2*time.Second, 2))
	if short.StreamNoteRatio != long.StreamNoteRatio {
		t.Fatal("length changed stream ratio")
	}
	if long.TappingConcentration >= concentrated.TappingConcentration {
		t.Fatal("distribution ignored")
	}
}
