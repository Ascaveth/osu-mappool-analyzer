package pattern

import (
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"math"
)

// computeIntensity uses actual timestamps, including local tempo changes.
func computeIntensity(bm *domain.Beatmap, p *SkillsetProfile, rate, cs float64) {
	p.TappingRate, p.TappingConcentration, p.AimSpeed = 0, 0, 0
	objects := orderedHitObjects(bm)
	if len(objects) < 2 {
		return
	}
	origin := objects[0].StartTime
	bins := map[int]float64{}
	total, intervals := 0.0, 0
	var previous *domain.HitObject
	diameter := math.Max(1, 108.8-8.96*cs)
	for i := range objects {
		h := &objects[i]
		if h.Type != domain.HitObjectCircle {
			previous = nil
			continue
		}
		if previous != nil {
			seconds := (h.StartTime - previous.StartTime).Seconds() / rate
			if seconds > 0 {
				tapping := 1 / seconds
				total += tapping
				intervals++
				bin := int((h.StartTime - origin).Seconds() / rate / 10)
				bins[bin] += tapping
				p.AimSpeed += distance(*previous, *h) / diameter / seconds
			}
		}
		previous = h
	}
	if intervals == 0 {
		return
	}
	p.TappingRate = total / float64(intervals)
	p.AimSpeed /= float64(intervals)
	peak := 0.0
	for _, value := range bins {
		if value > peak {
			peak = value
		}
	}
	p.TappingConcentration = peak / total
}
