package pattern

import (
	"context"
	"fmt"
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

	beatLength := time.Duration(60000.0/bm.BPM) * time.Millisecond
	snapThreshold := time.Duration(float64(beatLength) / streamSnapDivisor * snapToleranceRatio)

	burstCount, streamCount, longestRun := 0, 0, 0
	runLength := 1
	finalizeRun := func(length int) {
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

	for i := 1; i < len(circles); i++ {
		ioi := circles[i].StartTime - circles[i-1].StartTime
		if ioi <= snapThreshold {
			runLength++
			continue
		}
		finalizeRun(runLength)
		runLength = 1
	}
	finalizeRun(runLength)

	return analysis.Result{Metrics: map[string]float64{
		"burst_count":        float64(burstCount),
		"stream_count":       float64(streamCount),
		"longest_run_length": float64(longestRun),
	}}, nil
}

var _ analysis.Analyzer = StreamBurstAnalyzer{}
