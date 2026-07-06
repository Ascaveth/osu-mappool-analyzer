package pattern

import (
	"context"
	"fmt"
	"math"
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

	// deathstreamMinLength marks a run as "deathstream"-scale: the osu!
	// wiki calls unbroken streams at this length an endurance test rather
	// than a rhythm/reading test, ranked only as an explicit, approved
	// exception rather than by default. There is no official numeric
	// cutoff in the wiki itself, so 32 notes (two full bars of unbroken
	// 1/4-snap notes in 4/4) is this codebase's own stated convention,
	// same as streamMinLength/burstMinLength above — revisable, not a
	// fact about the beatmap.
	deathstreamMinLength = 32
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

	var findings []domain.Finding
	if longestRun >= deathstreamMinLength {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("longest unbroken run is %d notes, at or beyond the deathstream threshold (%d)", longestRun, deathstreamMinLength),
			Reason:         "the osu! wiki treats unbroken streams at this length as an endurance test rather than a rhythm/reading test, and ranks them only as an explicit, approved exception rather than by default",
			Recommendation: "confirm this run length is an intentional design choice for this slot rather than accidental overmapping",
		})
	}

	return analysis.Result{Metrics: map[string]float64{
		"burst_count":        float64(burstCount),
		"stream_count":       float64(streamCount),
		"longest_run_length": float64(longestRun),
	}, Findings: findings}, nil
}

// groupRuns groups consecutive circles into runs by inter-onset interval
// (IOI): a circle continues the current run when its IOI from the previous
// circle is at or under the local snap threshold, otherwise the run ends
// and a new one starts. It returns the circles of every run found, in
// order. Shared by StreamBurstAnalyzer and skillset classification (see
// ComputeSkillsetProfile) so the two stay consistent; callers apply their
// own length- or spacing-based classification on top.
func groupRuns(circles []domain.HitObject, sortedTimingPoints []domain.TimingPoint, fallbackBeatLengthMs float64) [][]domain.HitObject {
	if len(circles) == 0 {
		return nil
	}
	var runs [][]domain.HitObject
	current := []domain.HitObject{circles[0]}
	cursor := timingPointCursor{}
	for i := 1; i < len(circles); i++ {
		ioi := circles[i].StartTime - circles[i-1].StartTime
		beatLengthMs := cursor.beatLengthMs(sortedTimingPoints, circles[i-1].StartTime, fallbackBeatLengthMs)
		snapThreshold := time.Duration(beatLengthMs / streamSnapDivisor * snapToleranceRatio * float64(time.Millisecond))
		if ioi <= snapThreshold {
			current = append(current, circles[i])
			continue
		}
		runs = append(runs, current)
		current = []domain.HitObject{circles[i]}
	}
	runs = append(runs, current)
	return runs
}

// runLengths is a thin wrapper over groupRuns for callers that only need
// run lengths, not the underlying circles.
func runLengths(circles []domain.HitObject, sortedTimingPoints []domain.TimingPoint, fallbackBeatLengthMs float64) []int {
	groups := groupRuns(circles, sortedTimingPoints, fallbackBeatLengthMs)
	lengths := make([]int, len(groups))
	for i, g := range groups {
		lengths[i] = len(g)
	}
	return lengths
}

// spacingCV returns the coefficient of variation (stddev/mean) of
// straight-line spacing between consecutive circles in run — the signal
// that tells a clean, predictable jumpstream (low CV: the same spacing
// repeated, an aim/endurance test no different in kind from a stream slot
// that happens to have jumps in it) from an irregular one (high CV:
// spacing shifts note-to-note, a pattern-adaptation test using stream
// density as its vehicle). Returns 0 when run has fewer than two spacing
// samples, or when every sample is zero (fully stacked notes have no
// spacing to vary).
func spacingCV(run []domain.HitObject) float64 {
	if len(run) < 3 {
		return 0
	}
	distances := make([]float64, 0, len(run)-1)
	sum := 0.0
	for i := 1; i < len(run); i++ {
		d := distance(run[i-1], run[i])
		distances = append(distances, d)
		sum += d
	}
	mean := sum / float64(len(distances))
	if mean == 0 {
		return 0
	}
	variance := 0.0
	for _, d := range distances {
		diff := d - mean
		variance += diff * diff
	}
	variance /= float64(len(distances))
	return math.Sqrt(variance) / mean
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
