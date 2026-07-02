package pattern

import (
	"sort"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// SkillsetProfile is the set of raw, per-beatmap pattern primitives needed
// to classify a beatmap's dominant mechanical skillset(s) (aim, stream,
// tech, etc.). It mirrors the metrics JumpDistanceAnalyzer,
// StreamBurstAnalyzer, and SliderComplexityAnalyzer already compute and
// report independently — ComputeSkillsetProfile exists as a shared pure
// function (not an Analyzer) so tournament-level analyzers can classify
// skillsets without depending on this package's Analyzer implementations,
// which would violate the "analyzers never call each other" rule
// (docs/04-architecture-principles.md, Principle 3). This function reuses
// the same private helpers (landingObjects, distance, localBeatLengthMs)
// those analyzers already use, so the two computations cannot drift apart.
type SkillsetProfile struct {
	AvgJumpDistance    float64
	MaxJumpDistance    float64
	StreamCount        int
	LongestRunLength   int
	AvgAnchorCount     float64
	ReverseSliderRatio float64
	SpinnerDensity     float64
	ObjectCount        int
}

// ComputeSkillsetProfile derives a SkillsetProfile from a beatmap's raw hit
// object and timing data. It never panics on sparse or malformed input
// (zero BPM, empty HitObjects) — those cases simply produce a zero-value
// profile, matching the zero-count early returns every pattern Analyzer in
// this package already uses.
func ComputeSkillsetProfile(bm *domain.Beatmap) SkillsetProfile {
	profile := SkillsetProfile{ObjectCount: len(bm.HitObjects)}

	// Jump distance, reusing landingObjects/distance exactly as
	// JumpDistanceAnalyzer does.
	maxJump, sumJump, jumpCount := 0.0, 0.0, 0
	for _, objects := range landingObjects(bm) {
		for i := 1; i < len(objects); i++ {
			d := distance(objects[i-1], objects[i])
			if d > maxJump {
				maxJump = d
			}
			sumJump += d
			jumpCount++
		}
	}
	if jumpCount > 0 {
		profile.AvgJumpDistance = sumJump / float64(jumpCount)
		profile.MaxJumpDistance = maxJump
	}

	// Stream/burst run lengths, reusing localBeatLengthMs exactly as
	// StreamBurstAnalyzer does.
	if bm.BPM > 0 {
		var circles []domain.HitObject
		for _, h := range orderedHitObjects(bm) {
			if h.Type == domain.HitObjectCircle {
				circles = append(circles, h)
			}
		}
		if len(circles) >= burstMinLength {
			timingPoints := append([]domain.TimingPoint(nil), bm.TimingPoints...)
			sort.SliceStable(timingPoints, func(i, j int) bool { return timingPoints[i].Offset < timingPoints[j].Offset })
			fallbackBeatLengthMs := 60000.0 / bm.BPM

			streamCount, longestRun := 0, 0
			for _, length := range runLengths(circles, timingPoints, fallbackBeatLengthMs) {
				if length > longestRun {
					longestRun = length
				}
				if length >= streamMinLength {
					streamCount++
				}
			}
			profile.StreamCount = streamCount
			profile.LongestRunLength = longestRun
		}
	}

	// Slider complexity, reusing the same fields SliderComplexityAnalyzer reports.
	var sliders []domain.HitObject
	for _, h := range bm.HitObjects {
		if h.Type == domain.HitObjectSlider {
			sliders = append(sliders, h)
		}
	}
	if len(sliders) > 0 {
		anchorSum, reverseCount := 0, 0
		for _, s := range sliders {
			anchorSum += s.CurvePointCount
			if s.Repeats > 0 {
				reverseCount++
			}
		}
		profile.AvgAnchorCount = float64(anchorSum) / float64(len(sliders))
		profile.ReverseSliderRatio = float64(reverseCount) / float64(len(sliders))
	}

	// Spinner density, reusing the same fields SpinnerUsageAnalyzer reports.
	var spinners []domain.HitObject
	for _, h := range bm.HitObjects {
		if h.Type == domain.HitObjectSpinner {
			spinners = append(spinners, h)
		}
	}
	if len(spinners) > 0 && bm.LengthSeconds > 0 {
		totalDurationSeconds := 0.0
		for _, s := range spinners {
			if d := (s.EndTime - s.StartTime).Seconds(); d > 0 {
				totalDurationSeconds += d
			}
		}
		profile.SpinnerDensity = totalDurationSeconds / float64(bm.LengthSeconds)
	}

	return profile
}
