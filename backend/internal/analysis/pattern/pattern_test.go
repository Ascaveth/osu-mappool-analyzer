package pattern

import (
	"context"
	"testing"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func buildTournament(bm *domain.Beatmap) *domain.Tournament {
	return &domain.Tournament{
		ID:   "t-1",
		Name: "Test Open",
		Stages: []domain.Stage{
			{
				ID: "stage-1", Order: 1,
				Categories: []domain.Category{
					{
						ID: "cat-1", Order: 1,
						Slots: []domain.Slot{{ID: "slot-1", Position: 1, Beatmap: bm}},
					},
				},
			},
		},
	}
}

func scope(id string) domain.Scope { return domain.Scope{Type: domain.ScopeBeatmap, ID: id} }

func circle(x, y, t int) domain.HitObject {
	return domain.HitObject{Type: domain.HitObjectCircle, X: x, Y: y, StartTime: ms(t), EndTime: ms(t)}
}

func spinner(x, y, start, end int) domain.HitObject {
	return domain.HitObject{Type: domain.HitObjectSpinner, X: x, Y: y, StartTime: ms(start), EndTime: ms(end)}
}

func slider(x, y, t, anchors, repeats int) domain.HitObject {
	return domain.HitObject{
		Type: domain.HitObjectSlider, X: x, Y: y, StartTime: ms(t), EndTime: ms(t + 500),
		CurvePointCount: anchors, Repeats: repeats,
	}
}

// --- JumpDistanceAnalyzer ---

func TestJumpDistanceAnalyzer_ComputesDistances(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		circle(0, 0, 0),
		circle(100, 0, 100),
		circle(100, 100, 200),
	}}

	result, err := JumpDistanceAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["jump_count"]; got != 2 {
		t.Errorf("jump_count = %v, want 2", got)
	}
	if got := result.Metrics["jump_distance_avg"]; got != 100 {
		t.Errorf("jump_distance_avg = %v, want 100", got)
	}
}

func TestJumpDistanceAnalyzer_BreaksRunAtSpinners(t *testing.T) {
	// The cursor's exit position from a spinner is undefined (depends on
	// spin direction/timing, not map design), so the object after a
	// spinner must not be paired with the object before it: that single
	// circle before and the single circle after each form a run of one,
	// producing zero jumps, not a jump across the spinner.
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		circle(0, 0, 0),
		spinner(400, 300, 500, 900),
		circle(100, 0, 1000),
	}}

	result, err := JumpDistanceAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["jump_count"]; got != 0 {
		t.Fatalf("jump_count = %v, want 0 (spinner breaks the run, no jump spans it)", got)
	}
}

func TestJumpDistanceAnalyzer_FewerThanTwoObjects(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{circle(0, 0, 0)}}

	result, err := JumpDistanceAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["jump_count"]; got != 0 {
		t.Errorf("jump_count = %v, want 0", got)
	}
}

// --- JumpAngleAnalyzer ---

func TestJumpAngleAnalyzer_RightAngle(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		circle(0, 0, 0),
		circle(100, 0, 100),
		circle(100, 100, 200),
	}}

	result, err := JumpAngleAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["angle_count"]; got != 1 {
		t.Fatalf("angle_count = %v, want 1", got)
	}
	if got := result.Metrics["avg_angle_degrees"]; got < 89.9 || got > 90.1 {
		t.Errorf("avg_angle_degrees = %v, want ~90", got)
	}
	if got := result.Metrics["sharp_turn_count"]; got != 0 {
		t.Errorf("sharp_turn_count = %v, want 0 (90deg is not < threshold)", got)
	}
}

func TestJumpAngleAnalyzer_SkipsStackedNotes(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		circle(0, 0, 0),
		circle(0, 0, 100), // stacked on previous note: undefined angle
		circle(100, 0, 200),
	}}

	result, err := JumpAngleAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["angle_count"]; got != 0 {
		t.Errorf("angle_count = %v, want 0 (zero-length vector at the only interior point)", got)
	}
}

// --- StreamBurstAnalyzer ---

func TestStreamBurstAnalyzer_DetectsStream(t *testing.T) {
	// 180 BPM -> beatLength ~333ms -> 1/4 snap ~83ms; 80ms spacing qualifies.
	var objects []domain.HitObject
	for i := 0; i < 8; i++ {
		objects = append(objects, circle(i*10, 0, i*80))
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["stream_count"]; got != 1 {
		t.Errorf("stream_count = %v, want 1", got)
	}
	if got := result.Metrics["longest_run_length"]; got != 8 {
		t.Errorf("longest_run_length = %v, want 8", got)
	}
}

func TestStreamBurstAnalyzer_DetectsBurst(t *testing.T) {
	var objects []domain.HitObject
	for i := 0; i < 4; i++ {
		objects = append(objects, circle(i*10, 0, i*80))
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["burst_count"]; got != 1 {
		t.Errorf("burst_count = %v, want 1", got)
	}
	if got := result.Metrics["stream_count"]; got != 0 {
		t.Errorf("stream_count = %v, want 0", got)
	}
}

func TestStreamBurstAnalyzer_WidelySpacedNotesProduceNoRuns(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: []domain.HitObject{
		circle(0, 0, 0), circle(0, 0, 1000), circle(0, 0, 2000),
	}}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["burst_count"]; got != 0 {
		t.Errorf("burst_count = %v, want 0", got)
	}
	if got := result.Metrics["stream_count"]; got != 0 {
		t.Errorf("stream_count = %v, want 0", got)
	}
}

func TestStreamBurstAnalyzer_UsesLocalTimingNotGlobalBPM(t *testing.T) {
	// bm.BPM (dominant) reflects only the first, longer segment (120 BPM,
	// beatLength 500ms). The second segment slows to 60 BPM (beatLength
	// 1000ms) starting at 10000ms. Notes spaced 200ms apart are a stream
	// at 120 BPM (1/4 snap ~143ms threshold) but would NOT qualify at 60
	// BPM (1/4 snap ~287ms threshold still covers 200ms — pick a spacing
	// that distinguishes the two): use 220ms spacing, which is within the
	// 120 BPM threshold (~143ms) only if read against the wrong segment;
	// pick values that clearly differ between local and global lookup.
	bm := &domain.Beatmap{
		ID:  "bm-1",
		BPM: 120, // dominant/global BPM: the first, longer segment
		TimingPoints: []domain.TimingPoint{
			{Offset: ms(0), BeatLength: 500, Uninherited: true},      // 120 BPM
			{Offset: ms(10000), BeatLength: 1000, Uninherited: true}, // 60 BPM
		},
		HitObjects: []domain.HitObject{
			circle(0, 0, 10000),
			circle(10, 0, 10220),
			circle(20, 0, 10440),
			circle(30, 0, 10660),
		},
	}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	// At the active local tempo (60 BPM, beatLength 1000ms), 1/4 snap
	// tolerance is 1000/4*1.15 = 287.5ms, so 220ms spacing qualifies as
	// one run of 4 (a burst). Using the global 120 BPM beatLength
	// (500ms) instead, the tolerance would be 143.75ms, which 220ms
	// spacing would NOT satisfy, breaking the run into singletons.
	if got := result.Metrics["burst_count"]; got != 1 {
		t.Errorf("burst_count = %v, want 1 (local 60 BPM timing should classify this as one run)", got)
	}
}

func TestStreamBurstAnalyzer_FlagsDeathstream(t *testing.T) {
	var objects []domain.HitObject
	for i := 0; i < deathstreamMinLength; i++ {
		objects = append(objects, circle(i*10, 0, i*80))
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1 (run length %d meets deathstream threshold %d)", len(result.Findings), deathstreamMinLength, deathstreamMinLength)
	}
	if result.Findings[0].Severity != domain.SeverityWarning {
		t.Errorf("Severity = %v, want Warning", result.Findings[0].Severity)
	}
}

func TestStreamBurstAnalyzer_BelowDeathstreamThresholdNoFinding(t *testing.T) {
	var objects []domain.HitObject
	for i := 0; i < deathstreamMinLength-1; i++ {
		objects = append(objects, circle(i*10, 0, i*80))
	}
	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: objects}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0 (run one note short of the deathstream threshold)", len(result.Findings))
	}
}

func TestStreamBurstAnalyzer_ZeroBPMDoesNotPanic(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", BPM: 0, HitObjects: []domain.HitObject{
		circle(0, 0, 0), circle(0, 0, 80), circle(0, 0, 160),
	}}

	result, err := StreamBurstAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["stream_count"]; got != 0 {
		t.Errorf("stream_count = %v, want 0", got)
	}
}

// --- SliderComplexityAnalyzer ---

func TestSliderComplexityAnalyzer_ComputesMetricsAndFlagsMalformed(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{
		slider(0, 0, 0, 2, 1),
		slider(100, 0, 1000, 0, 0), // malformed: zero anchors
	}}

	result, err := SliderComplexityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["slider_count"]; got != 2 {
		t.Errorf("slider_count = %v, want 2", got)
	}
	if got := result.Metrics["avg_anchor_count"]; got != 1 {
		t.Errorf("avg_anchor_count = %v, want 1", got)
	}
	if got := result.Metrics["reverse_slider_ratio"]; got != 0.5 {
		t.Errorf("reverse_slider_ratio = %v, want 0.5", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
	if result.Findings[0].Severity != domain.SeverityWarning {
		t.Errorf("Severity = %v, want Warning", result.Findings[0].Severity)
	}
}

func TestSliderComplexityAnalyzer_NoSliders(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{circle(0, 0, 0)}}

	result, err := SliderComplexityAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}

// --- SpinnerUsageAnalyzer ---

func TestSpinnerUsageAnalyzer_ComputesDensityAndFlagsInvalid(t *testing.T) {
	bm := &domain.Beatmap{
		ID: "bm-1", LengthSeconds: 10,
		HitObjects: []domain.HitObject{
			spinner(256, 192, 0, 2000),    // valid 2s spinner
			spinner(256, 192, 5000, 5000), // invalid: zero duration
		},
	}

	result, err := SpinnerUsageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if got := result.Metrics["spinner_count"]; got != 2 {
		t.Errorf("spinner_count = %v, want 2", got)
	}
	if got := result.Metrics["total_spinner_duration_seconds"]; got != 2 {
		t.Errorf("total_spinner_duration_seconds = %v, want 2", got)
	}
	if got := result.Metrics["spinner_density"]; got != 0.2 {
		t.Errorf("spinner_density = %v, want 0.2", got)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Findings))
	}
}

func TestSpinnerUsageAnalyzer_NoSpinners(t *testing.T) {
	bm := &domain.Beatmap{ID: "bm-1", HitObjects: []domain.HitObject{circle(0, 0, 0)}}

	result, err := SpinnerUsageAnalyzer{}.Analyze(context.Background(), analysis.Input{
		Tournament: buildTournament(bm), Scope: scope(bm.ID),
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(result.Findings))
	}
}

// --- Integration ---

func TestPatternAnalyzers_RunTogetherInEngine(t *testing.T) {
	e := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		JumpDistanceAnalyzer{}, JumpAngleAnalyzer{}, StreamBurstAnalyzer{},
		SliderComplexityAnalyzer{}, SpinnerUsageAnalyzer{},
	} {
		if err := e.Register(a); err != nil {
			t.Fatalf("Register(%s): %v", a.Name(), err)
		}
	}

	bm := &domain.Beatmap{ID: "bm-1", BPM: 180, HitObjects: []domain.HitObject{
		circle(0, 0, 0), slider(100, 0, 200, 2, 0), spinner(256, 192, 2000, 3000),
	}}

	results, err := e.Run(context.Background(), buildTournament(bm))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("len(results) = %d, want 5 (one per beatmap-scoped analyzer)", len(results))
	}
}
