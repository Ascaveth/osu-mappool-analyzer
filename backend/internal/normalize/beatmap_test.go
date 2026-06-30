package normalize

import (
	"bytes"
	"os"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osufile"
)

func parseTestdata(t *testing.T, path string) ([]byte, *osufile.RawBeatmap) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading testdata: %v", err)
	}
	raw, err := osufile.Parse(bytes.NewReader(source))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return source, raw
}

func TestBeatmap_HappyPath(t *testing.T) {
	source, raw := parseTestdata(t, "../osufile/testdata/sample.osu")

	bm, err := Beatmap(raw, source)
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}

	if bm.Title != "Test Song" {
		t.Errorf("Title = %q, want %q", bm.Title, "Test Song")
	}
	if bm.Mapper != "Tester" {
		t.Errorf("Mapper = %q, want %q", bm.Mapper, "Tester")
	}
	if bm.AR != 9 || bm.OD != 8 || bm.CS != 4 || bm.HP != 5 {
		t.Errorf("difficulty = AR%.0f OD%.0f CS%.0f HP%.0f, want AR9 OD8 CS4 HP5", bm.AR, bm.OD, bm.CS, bm.HP)
	}

	if bm.ObjectCount != 3 {
		t.Fatalf("ObjectCount = %d, want 3", bm.ObjectCount)
	}
	wantSliderRatio := 1.0 / 3.0
	if diff := bm.SliderRatio - wantSliderRatio; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("SliderRatio = %v, want %v", bm.SliderRatio, wantSliderRatio)
	}

	// 120 BPM covers offset 0-20000ms (20s of playtime); 200 BPM only
	// covers 20000-27000ms (7s) — 120 BPM should dominate.
	if bm.BPM != 120 {
		t.Errorf("BPM = %v, want 120 (dominant segment)", bm.BPM)
	}

	// Earliest object start = circle at 1000ms; latest object end =
	// spinner end at 27000ms -> 26s.
	if bm.LengthSeconds != 26 {
		t.Errorf("LengthSeconds = %d, want 26", bm.LengthSeconds)
	}

	if bm.OsuFileHash == "" {
		t.Error("OsuFileHash should not be empty")
	}

	// Slider duration: pixelLength=140, slides=2, beatLength=500,
	// velocity=1.0 (no inherited point precedes it), sliderMultiplier=1.4
	// -> pixelsPerBeat=140 -> beats=2 -> duration=1000ms -> end=3000ms.
	slider := bm.HitObjects[1]
	wantEndMs := int64(3000)
	if gotEndMs := slider.EndTime.Milliseconds(); gotEndMs != wantEndMs {
		t.Errorf("slider EndTime = %dms, want %dms", gotEndMs, wantEndMs)
	}
}

func TestBeatmap_SliderCurvePointCount(t *testing.T) {
	source, raw := parseTestdata(t, "../osufile/testdata/sample.osu")

	bm, err := Beatmap(raw, source)
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}

	// sample.osu's slider curve data is "B|250:200" — one anchor point.
	slider := bm.HitObjects[1]
	if slider.CurvePointCount != 1 {
		t.Errorf("CurvePointCount = %d, want 1", slider.CurvePointCount)
	}
}

func TestBeatmap_MissingRequiredDifficultyField(t *testing.T) {
	source, raw := parseTestdata(t, "../osufile/testdata/missing_difficulty.osu")

	_, err := Beatmap(raw, source)
	if err == nil {
		t.Fatal("Beatmap should return an error when a required [Difficulty] field is missing")
	}
}

func TestBeatmap_InvalidApproachRateFailsFast(t *testing.T) {
	input := `osu file format v14

[Difficulty]
HPDrainRate:5
CircleSize:4
OverallDifficulty:5
ApproachRate:not-a-number

[TimingPoints]
0,500,4,2,0,100,1,0
`
	raw, err := osufile.Parse(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if _, err := Beatmap(raw, []byte(input)); err == nil {
		t.Fatal("Beatmap should return an error when ApproachRate is present but unparseable, not silently fall back to OverallDifficulty")
	}
}

func TestBeatmap_SameSourceProducesSameHash(t *testing.T) {
	source, raw := parseTestdata(t, "../osufile/testdata/sample.osu")

	a, err := Beatmap(raw, source)
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}
	b, err := Beatmap(raw, source)
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}
	if a.OsuFileHash != b.OsuFileHash {
		t.Errorf("hash not deterministic: %q != %q", a.OsuFileHash, b.OsuFileHash)
	}
}

func TestBeatmap_ExtremeBPMAndDenseObjectsNormalizeWithoutClamping(t *testing.T) {
	source, raw := parseTestdata(t, "../osufile/testdata/extreme_values.osu")

	bm, err := Beatmap(raw, source)
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}

	// Second timing point (90ms beat length = 666.67 BPM) covers the
	// longer segment (60000ms-180000ms vs the first's 0-60000ms), so it's
	// dominant. Either way, normalize must report a triple-digit BPM
	// as-is, not clamp or reject an unusually high value.
	if bm.BPM != 666.67 {
		t.Errorf("BPM = %v, want 666.67", bm.BPM)
	}
	if bm.ObjectCount != 6 {
		t.Fatalf("ObjectCount = %d, want 6", bm.ObjectCount)
	}
	// Earliest object starts at 0ms; spinner ends at 180000ms -> 180s.
	if bm.LengthSeconds != 180 {
		t.Errorf("LengthSeconds = %d, want 180", bm.LengthSeconds)
	}
}

func TestBeatmap_DominantBPMUsesTrueLatestHitObjectEndTime(t *testing.T) {
	// A spinner starting at 4000ms ends at 20000ms, but a circle that
	// starts later (4500ms, and so is the LAST element in file/StartTime
	// order) ends earlier, at 4500ms. The second uninherited timing
	// point's segment must be measured against the true latest end time
	// (20000ms, from the spinner) rather than the last hit object in
	// slice order (4500ms) — otherwise the second segment is measured as
	// having ~0 duration and the dominant-BPM selection picks the wrong
	// segment.
	input := `osu file format v14

[Difficulty]
HPDrainRate:5
CircleSize:4
OverallDifficulty:5

[TimingPoints]
0,1000,4,2,0,100,1,0
5000,100,4,2,0,100,1,0

[HitObjects]
256,192,4000,8,0,20000,0:0:0:0:
100,100,4500,1,0,0:0:0:0:
`
	raw, err := osufile.Parse(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	bm, err := Beatmap(raw, []byte(input))
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}

	// First segment (0-5000ms, 60 BPM) covers 5000ms. Second segment
	// (5000-20000ms, 600 BPM) covers 15000ms once measured against the
	// spinner's true end time, so 600 BPM should dominate.
	if bm.BPM != 600 {
		t.Errorf("BPM = %v, want 600 (dominant segment measured against the spinner's true end time)", bm.BPM)
	}
}

func TestBeatmap_MissingTimingPointsYieldsZeroBPM(t *testing.T) {
	source, raw := parseTestdata(t, "../osufile/testdata/missing_timing_points.osu")

	bm, err := Beatmap(raw, source)
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}
	// No uninherited timing point exists to derive a dominant BPM from.
	if bm.BPM != 0 {
		t.Errorf("BPM = %v, want 0 (no timing points to derive it from)", bm.BPM)
	}
	if bm.ObjectCount != 2 {
		t.Errorf("ObjectCount = %d, want 2 (hit objects still normalize without timing data)", bm.ObjectCount)
	}
}

func TestBeatmap_EmptyHitObjectsDoesNotPanic(t *testing.T) {
	input := `osu file format v14

[Difficulty]
HPDrainRate:5
CircleSize:4
OverallDifficulty:5

[TimingPoints]
0,500,4,2,0,100,1,0
`
	raw, err := osufile.Parse(bytes.NewReader([]byte(input)))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	bm, err := Beatmap(raw, []byte(input))
	if err != nil {
		t.Fatalf("Beatmap returned error: %v", err)
	}
	if bm.ObjectCount != 0 || bm.LengthSeconds != 0 || bm.SliderRatio != 0 {
		t.Errorf("expected zero-value metrics for empty pool, got ObjectCount=%d LengthSeconds=%d SliderRatio=%v",
			bm.ObjectCount, bm.LengthSeconds, bm.SliderRatio)
	}
}
