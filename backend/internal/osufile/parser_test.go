package osufile

import (
	"os"
	"strings"
	"testing"
)

func TestParse_HappyPath(t *testing.T) {
	f, err := os.Open("testdata/sample.osu")
	if err != nil {
		t.Fatalf("opening testdata: %v", err)
	}
	defer f.Close()

	raw, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if raw.FormatVersion != 14 {
		t.Errorf("FormatVersion = %d, want 14", raw.FormatVersion)
	}
	if got := raw.Metadata["Title"]; got != "Test Song" {
		t.Errorf("Metadata[Title] = %q, want %q", got, "Test Song")
	}
	if got := raw.Metadata["Creator"]; got != "Tester" {
		t.Errorf("Metadata[Creator] = %q, want %q", got, "Tester")
	}
	if got := raw.Difficulty["CircleSize"]; got != "4" {
		t.Errorf("Difficulty[CircleSize] = %q, want %q", got, "4")
	}

	if len(raw.TimingPoints) != 3 {
		t.Fatalf("len(TimingPoints) = %d, want 3", len(raw.TimingPoints))
	}
	if !raw.TimingPoints[0].Uninherited || raw.TimingPoints[0].BeatLength != 500 {
		t.Errorf("TimingPoints[0] = %+v, want uninherited BeatLength=500", raw.TimingPoints[0])
	}
	if raw.TimingPoints[1].Uninherited {
		t.Errorf("TimingPoints[1] should be inherited, got %+v", raw.TimingPoints[1])
	}

	if len(raw.HitObjects) != 3 {
		t.Fatalf("len(HitObjects) = %d, want 3", len(raw.HitObjects))
	}
	if raw.HitObjects[0].Type != RawHitObjectCircle {
		t.Errorf("HitObjects[0].Type = %v, want circle", raw.HitObjects[0].Type)
	}
	if raw.HitObjects[1].Type != RawHitObjectSlider {
		t.Errorf("HitObjects[1].Type = %v, want slider", raw.HitObjects[1].Type)
	}
	if raw.HitObjects[1].Slides != 2 {
		t.Errorf("HitObjects[1].Slides = %d, want 2", raw.HitObjects[1].Slides)
	}
	if raw.HitObjects[1].SliderLength != 140 {
		t.Errorf("HitObjects[1].SliderLength = %v, want 140", raw.HitObjects[1].SliderLength)
	}
	if raw.HitObjects[2].Type != RawHitObjectSpinner {
		t.Errorf("HitObjects[2].Type = %v, want spinner", raw.HitObjects[2].Type)
	}
	if raw.HitObjects[2].EndTime != 27000 {
		t.Errorf("HitObjects[2].EndTime = %v, want 27000", raw.HitObjects[2].EndTime)
	}
}

func TestParse_NotAnOsuFile(t *testing.T) {
	_, err := Parse(strings.NewReader("this is not a beatmap\njust some text\n"))
	if err != ErrNotAnOsuFile {
		t.Fatalf("Parse error = %v, want ErrNotAnOsuFile", err)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err != ErrNotAnOsuFile {
		t.Fatalf("Parse error = %v, want ErrNotAnOsuFile", err)
	}
}

func TestParse_SkipsMalformedLinesWithoutFailing(t *testing.T) {
	input := `osu file format v14

[TimingPoints]
not,a,valid,line
0,500,4,2,0,100,1,0

[HitObjects]
also,not,valid
100,100,1000,1,0,0:0:0:0:
`
	raw, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(raw.TimingPoints) != 1 {
		t.Errorf("len(TimingPoints) = %d, want 1 (malformed line skipped)", len(raw.TimingPoints))
	}
	if len(raw.HitObjects) != 1 {
		t.Errorf("len(HitObjects) = %d, want 1 (malformed line skipped)", len(raw.HitObjects))
	}
}

func TestParse_BOMHeaderIsTolerated(t *testing.T) {
	input := "\uFEFFosu file format v14\n\n[Metadata]\nTitle:BOM Test\n"
	raw, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if raw.Metadata["Title"] != "BOM Test" {
		t.Errorf("Metadata[Title] = %q, want %q", raw.Metadata["Title"], "BOM Test")
	}
}

func TestParse_SliderCurvesAndInheritedTimingPoints(t *testing.T) {
	f, err := os.Open("testdata/slider_curves.osu")
	if err != nil {
		t.Fatalf("opening testdata: %v", err)
	}
	defer f.Close()

	raw, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(raw.TimingPoints) != 3 {
		t.Fatalf("len(TimingPoints) = %d, want 3", len(raw.TimingPoints))
	}
	if !raw.TimingPoints[0].Uninherited {
		t.Error("TimingPoints[0] should be uninherited (the red line)")
	}
	if raw.TimingPoints[1].Uninherited || raw.TimingPoints[1].BeatLength >= 0 {
		t.Errorf("TimingPoints[1] should be an inherited (green) line with negative BeatLength, got %+v", raw.TimingPoints[1])
	}

	if len(raw.HitObjects) != 3 {
		t.Fatalf("len(HitObjects) = %d, want 3", len(raw.HitObjects))
	}

	bezier := raw.HitObjects[1]
	if bezier.Type != RawHitObjectSlider {
		t.Fatalf("HitObjects[1].Type = %v, want slider", bezier.Type)
	}
	if bezier.CurveType != "B" || bezier.CurvePointCount != 3 {
		t.Errorf("bezier slider: CurveType=%q CurvePointCount=%d, want \"B\" and 3 anchors", bezier.CurveType, bezier.CurvePointCount)
	}
	if bezier.Slides != 3 {
		t.Errorf("bezier slider: Slides = %d, want 3", bezier.Slides)
	}

	perfectCircle := raw.HitObjects[2]
	if perfectCircle.CurveType != "P" || perfectCircle.CurvePointCount != 2 {
		t.Errorf("perfect-circle slider: CurveType=%q CurvePointCount=%d, want \"P\" and 2 anchors", perfectCircle.CurveType, perfectCircle.CurvePointCount)
	}
}

func TestParse_ExtremeValues(t *testing.T) {
	f, err := os.Open("testdata/extreme_values.osu")
	if err != nil {
		t.Fatalf("opening testdata: %v", err)
	}
	defer f.Close()

	raw, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if raw.Difficulty["OverallDifficulty"] != "10" || raw.Difficulty["ApproachRate"] != "10" {
		t.Errorf("Difficulty = %+v, want OD10/AR10", raw.Difficulty)
	}
	if len(raw.HitObjects) != 6 {
		t.Fatalf("len(HitObjects) = %d, want 6 (5 dense circles + 1 long spinner)", len(raw.HitObjects))
	}

	// A 100ms beat length is 600 BPM \u2014 the parser must not reject or clamp
	// an unusually high BPM, only normalize/analysis layers interpret it.
	if !raw.TimingPoints[0].Uninherited || raw.TimingPoints[0].BeatLength != 100 {
		t.Errorf("TimingPoints[0] = %+v, want uninherited BeatLength=100 (600 BPM)", raw.TimingPoints[0])
	}

	spinner := raw.HitObjects[5]
	if spinner.Type != RawHitObjectSpinner || spinner.EndTime != 180000 {
		t.Errorf("spinner = %+v, want EndTime=180000 (a 60s spinner)", spinner)
	}
}

func TestParse_MissingTimingPointsSectionDoesNotFail(t *testing.T) {
	f, err := os.Open("testdata/missing_timing_points.osu")
	if err != nil {
		t.Fatalf("opening testdata: %v", err)
	}
	defer f.Close()

	raw, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(raw.TimingPoints) != 0 {
		t.Errorf("len(TimingPoints) = %d, want 0", len(raw.TimingPoints))
	}
	if len(raw.HitObjects) != 2 {
		t.Errorf("len(HitObjects) = %d, want 2 (a missing [TimingPoints] section should not affect [HitObjects] parsing)", len(raw.HitObjects))
	}
}
