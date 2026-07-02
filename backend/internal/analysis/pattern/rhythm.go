package pattern

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// Stream/burst classification follows the convention commonly used in the
// osu! mapping community: a run of circles spaced at 1/4 beat (16th-note)
// snap or denser is a "stream" once it reaches 7+ notes, and a "burst"
// for shorter runs of 3-6 notes. This is a stated convention for
// explainability, not an objective fact about the beatmap — a future
// analyzer is free to reclassify with a different convention without
// this one's output being "wrong," since the metrics (run lengths) are
// reported regardless of classification.
const (
	streamSnapDivisor  = 4.0  // 1/4 beat = 16th-note snap
	snapToleranceRatio = 1.15 // allow 15% timing slack before breaking a run
	burstMinLength     = 3
	streamMinLength    = 7
)

// StreamBurstAnalyzer detects runs of closely-spaced circles and
// classifies them as bursts or streams by length. It reports neutral
// metrics only — detecting a stream is not itself a finding, since
// streams are a normal, often deliberate pattern choice; whether a pool
// has "enough" or "too many" streams is a Balance/Diversity judgment
// (Phase 8), not something this analyzer asserts.
type StreamBurstAnalyzer struct{}

func (StreamBurstAnalyzer) Name() string { return "stream-burst-analyzer" }

func (StreamBurstAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeBeatmap }

func (StreamBurstAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	bm := analysis.FindBeatmap(in.Tournament, in.Scope.ID)
	if bm == nil {
		return analysis.Result{}, fmt.Errorf("pattern: beatmap %q not found in tournament", in.Scope.ID)
	}

	zero := map[string]float64{"burst_count": 0, "stream_count": 0, "longest_run_length": 0}

	if bm.BPM <= 0 {
		return analysis.Result{Metrics: zero}, nil
	}

	var circles []domain.HitObject
	for _, h := range orderedHitObjects(bm) {
		if h.Type == domain.HitObjectCircle {
			circles = append(circles, h)
		}
	}
	if len(circles) < burstMinLength {
		return analysis.Result{Metrics: zero}, nil
	}

	timingPoints := append([]domain.TimingPoint(nil), bm.TimingPoints...)
	sort.SliceStable(timingPoints, func(i, j int) bool { return timingPoints[i].Offset < timingPoints[j].Offset })

	// fallbackBeatLengthMs covers fixtures/maps with no usable uninherited
	// timing point preceding a given object (e.g. synthetic test data that
	// only sets BPM directly): fall back to the beatmap's dominant BPM
	// rather than treating the interval as having no active tempo.
	fallbackBeatLengthMs := 60000.0 / bm.BPM

	burstCount, streamCount, longestRun := 0, 0, 0
	for _, length := range runLengths(circles, timingPoints, fallbackBeatLengthMs) {
		if length > longestRun {
			longestRun = length
		}
		switch {
		case length >= streamMinLength:
			streamCount++
		case length >= burstMinLength:
			burstCount++
		}
	}

	return analysis.Result{Metrics: map[string]float64{
		"burst_count":        float64(burstCount),
		"stream_count":       float64(streamCount),
		"longest_run_length": float64(longestRun),
	}}, nil
}

// runLengths groups consecutive circles into runs by inter-onset interval
// (IOI): a circle continues the current run when its IOI from the previous
// circle is at or under the local snap threshold, otherwise the run ends
// and a new one starts. It returns the length of every run found, in order.
// Shared by StreamBurstAnalyzer and skillset classification
// (see ComputeSkillsetProfile) so the two stay consistent; callers apply
// their own length-to-classification thresholds.
func runLengths(circles []domain.HitObject, sortedTimingPoints []domain.TimingPoint, fallbackBeatLengthMs float64) []int {
	if len(circles) == 0 {
		return nil
	}
	var lengths []int
	runLength := 1
	cursor := timingPointCursor{}
	for i := 1; i < len(circles); i++ {
		ioi := circles[i].StartTime - circles[i-1].StartTime
		beatLengthMs := cursor.beatLengthMs(sortedTimingPoints, circles[i-1].StartTime, fallbackBeatLengthMs)
		snapThreshold := time.Duration(beatLengthMs / streamSnapDivisor * snapToleranceRatio * float64(time.Millisecond))
		if ioi <= snapThreshold {
			runLength++
			continue
		}
		lengths = append(lengths, runLength)
		runLength = 1
	}
	lengths = append(lengths, runLength)
	return lengths
}

// timingPointCursor tracks the active uninherited beat length while
// scanning timing points in time order, so repeated lookups over a
// monotonically increasing sequence of query times (as runLengths performs)
// advance through the timing points once instead of rescanning from the
// start each time.
type timingPointCursor struct {
	idx        int
	beatLength float64
}

// beatLengthMs returns the beat length (ms) of the uninherited timing point
// active at time t — the most recent uninherited point at or before t in
// sortedPoints (sorted ascending by Offset). Falls back to fallbackMs when
// no such point has been seen (e.g. t precedes every timing point, or the
// beatmap has none at all).
//
// Callers must invoke this with non-decreasing t across the cursor's
// lifetime (as runLengths does, since circles are processed in time order);
// the cursor only advances forward and never rescans earlier points.
func (c *timingPointCursor) beatLengthMs(sortedPoints []domain.TimingPoint, t time.Duration, fallbackMs float64) float64 {
	for c.idx < len(sortedPoints) && sortedPoints[c.idx].Offset <= t {
		p := sortedPoints[c.idx]
		if p.Uninherited && p.BeatLength > 0 {
			c.beatLength = p.BeatLength
		}
		c.idx++
	}
	if c.beatLength > 0 {
		return c.beatLength
	}
	return fallbackMs
}

var _ analysis.Analyzer = StreamBurstAnalyzer{}
