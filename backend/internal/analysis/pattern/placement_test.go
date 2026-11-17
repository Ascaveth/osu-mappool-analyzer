package pattern

import (
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"reflect"
	"testing"
	"time"
)

func TestPlacementProfile(t *testing.T) {
	bm := &domain.Beatmap{BPM: 140, LengthSeconds: 60, HitObjects: []domain.HitObject{{StartTime: time.Second}}, TimingPoints: []domain.TimingPoint{{BeatLength: 60000.0 / 140}}}
	before := *bm
	before.HitObjects = append([]domain.HitObject(nil), bm.HitObjects...)
	before.TimingPoints = append([]domain.TimingPoint(nil), bm.TimingPoints...)
	for _, tc := range []struct {
		name string
		bpm  float64
		ok   bool
	}{{"NM", 140, true}, {"DT", 210, true}, {"HDDTHR", 210, true}, {"EZDT", 210, true}, {"HT", 105, true}, {"HD", 140, true}, {"FM", 0, false}, {"TB", 0, false}, {"custom", 0, false}, {"HREZ", 0, false}, {"DTHT", 0, false}} {
		p, ok := ComputePlacementSkillsetProfile(bm, tc.name)
		if ok != tc.ok || p.BPM != tc.bpm {
			t.Errorf("%s: %+v, %v", tc.name, p, ok)
		}
	}
	if !reflect.DeepEqual(*bm, before) {
		t.Fatal("source mutated")
	}
}
