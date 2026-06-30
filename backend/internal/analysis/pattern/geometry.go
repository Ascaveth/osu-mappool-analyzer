package pattern

import (
	"context"
	"fmt"
	"math"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// landingObjects returns circles and slider starts, in time order, split
// into runs that break at every spinner. A spinner has no fixed "landing
// position" — the cursor leaves it at an arbitrary point determined by
// spin direction and timing, not by map design — so a jump computed
// between the object before a spinner and the object after it would
// describe a transition that doesn't exist on the map. Each returned run
// is a maximal stretch of circles/slider-starts with no spinner inside
// it; jump distance/angle are only ever computed within one run, never
// across the break a spinner introduces.
func landingObjects(bm *domain.Beatmap) [][]domain.HitObject {
	var runs [][]domain.HitObject
	var current []domain.HitObject
	for _, h := range orderedHitObjects(bm) {
		if h.Type == domain.HitObjectSpinner {
			if len(current) > 0 {
				runs = append(runs, current)
				current = nil
			}
			continue
		}
		current = append(current, h)
	}
	if len(current) > 0 {
		runs = append(runs, current)
	}
	return runs
}

// distance computes the straight-line distance between two hit objects' start positions.
func distance(a, b domain.HitObject) float64 {
	dx := float64(b.X - a.X)
	dy := float64(b.Y - a.Y)
	return math.Hypot(dx, dy)
}

// JumpDistanceAnalyzer reports the straight-line distance between
// consecutive landing objects' start positions, in osu!pixels. It makes
// no judgment about whether a given jump distance is appropriate for a
// tournament's intended difficulty — that depends on context (BPM,
// approach rate, surrounding patterns) this analyzer doesn't have. It
// only describes what's there.
type JumpDistanceAnalyzer struct{}

func (JumpDistanceAnalyzer) Name() string { return "jump-distance-analyzer" }

func (JumpDistanceAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (JumpDistanceAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("pattern: beatmap %q not found in tournament", in.Scope.ID)
	}

	runs := landingObjects(bm)

	min, max, sum, count := math.Inf(1), 0.0, 0.0, 0
	for _, objects := range runs {
		for i := 1; i < len(objects); i++ {
			d := distance(objects[i-1], objects[i])
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
			sum += d
			count++
		}
	}
	if count == 0 {
		return analysis.Result{Metrics: map[string]float64{"jump_count": 0}}, nil
	}

	return analysis.Result{Metrics: map[string]float64{
		"jump_count":        float64(count),
		"jump_distance_min": min,
		"jump_distance_max": max,
		"jump_distance_avg": sum / float64(count),
	}}, nil
}

var _ analysis.Analyzer = JumpDistanceAnalyzer{}

// angleBetween returns the angle in degrees at point b, between vectors
// b->a and b->c. Returns false (undefined) if either vector has zero
// length, which happens when consecutive objects are stacked at the same
// position — a common, valid pattern (e.g. within a stream) that simply
// angleBetween returns the turning angle at b between the segments b→a and b→c.
// It reports false when either segment has zero length.
func angleBetween(a, b, c domain.HitObject) (float64, bool) {
	v1x, v1y := float64(a.X-b.X), float64(a.Y-b.Y)
	v2x, v2y := float64(c.X-b.X), float64(c.Y-b.Y)
	len1 := math.Hypot(v1x, v1y)
	len2 := math.Hypot(v2x, v2y)
	if len1 == 0 || len2 == 0 {
		return 0, false
	}
	cos := (v1x*v2x + v1y*v2y) / (len1 * len2)
	cos = math.Max(-1, math.Min(1, cos)) // clamp for float rounding
	return math.Acos(cos) * 180 / math.Pi, true
}

// JumpAngleAnalyzer reports the angle of direction change at each
// interior landing object — how sharply the cursor must turn between
// consecutive jumps. Like JumpDistanceAnalyzer, it describes the geometry
// without judging it: a "sharp turn" isn't inherently good or bad, it's a
// pattern characteristic a Balance or Diversity analyzer (Phase 8) can
// weigh against the rest of a category.
type JumpAngleAnalyzer struct{}

func (JumpAngleAnalyzer) Name() string { return "jump-angle-analyzer" }

func (JumpAngleAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

// sharpTurnThresholdDegrees marks an angle as a "sharp turn" when the
// cursor reverses more than it continues forward — i.e. the angle at the
// turning point is acute. This is a geometric fact (acute vs. obtuse),
// not a difficulty judgment.
const sharpTurnThresholdDegrees = 90.0

func (JumpAngleAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("pattern: beatmap %q not found in tournament", in.Scope.ID)
	}

	runs := landingObjects(bm)

	sum, count, sharpTurns := 0.0, 0, 0
	for _, objects := range runs {
		for i := 1; i < len(objects)-1; i++ {
			angle, ok := angleBetween(objects[i-1], objects[i], objects[i+1])
			if !ok {
				continue
			}
			sum += angle
			count++
			if angle < sharpTurnThresholdDegrees {
				sharpTurns++
			}
		}
	}

	metrics := map[string]float64{"angle_count": float64(count), "sharp_turn_count": float64(sharpTurns)}
	if count > 0 {
		metrics["avg_angle_degrees"] = sum / float64(count)
	}

	return analysis.Result{Metrics: metrics}, nil
}

var _ analysis.Analyzer = JumpAngleAnalyzer{}
