// Package normalize converts raw, format-faithful osufile.RawBeatmap data
// into the domain.Beatmap representation the Analysis Engine consumes.
// This is the "Normalization" stage of the Raw Data -> Normalization ->
// Analyzer pipeline (docs/04-architecture-principles.md).
package normalize

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osufile"
)

// defaultBeatLengthMs is used when a beatmap has no uninherited timing
// point preceding a hit object — this should not happen in a valid .osu
// file, but normalization must not panic on malformed input.
const defaultBeatLengthMs = 500.0 // 120 BPM

// Beatmap converts a parsed .osu file plus its original source bytes into
// a domain.Beatmap. The original bytes are required (rather than just the
// parsed struct) so OsuFileHash reflects the exact source file, including
// formatting differences a re-parse would normalize away.
func Beatmap(raw *osufile.RawBeatmap, sourceBytes []byte) (*domain.Beatmap, error) {
	if raw == nil {
		return nil, fmt.Errorf("normalize: raw beatmap is nil")
	}

	difficulty, err := parseDifficulty(raw.Difficulty, raw.FormatVersion)
	if err != nil {
		return nil, fmt.Errorf("normalize: %w", err)
	}

	timingPoints := normalizeTimingPoints(raw.TimingPoints)
	hitObjects, sliderCount := normalizeHitObjects(raw.HitObjects, raw.TimingPoints, parseFloatOr(raw.Difficulty["SliderMultiplier"], 1.0))

	bm := &domain.Beatmap{
		Title:        raw.Metadata["Title"],
		Artist:       raw.Metadata["Artist"],
		Mapper:       raw.Metadata["Creator"],
		Version:      raw.Metadata["Version"],
		Tags:         splitTags(raw.Metadata["Tags"]),
		AR:           difficulty.ar,
		OD:           difficulty.od,
		CS:           difficulty.cs,
		HP:           difficulty.hp,
		BPM:          dominantBPM(raw.TimingPoints, hitObjects),
		TimingPoints: timingPoints,
		HitObjects:   hitObjects,
		ObjectCount:  len(hitObjects),
		OsuFileHash:  hashSource(sourceBytes),
	}

	if bm.ObjectCount > 0 {
		bm.SliderRatio = float64(sliderCount) / float64(bm.ObjectCount)
		bm.LengthSeconds = lengthSeconds(hitObjects)
	}

	return bm, nil
}

type difficultySettings struct {
	ar, od, cs, hp float64
}

// parseDifficulty reads HP/CS/OD/AR. AR was introduced in format v8; older
// maps inherit AR from OD, matching osu!'s own documented fallback.
func parseDifficulty(d map[string]string, formatVersion int) (difficultySettings, error) {
	hp, err := requireFloat(d, "HPDrainRate")
	if err != nil {
		return difficultySettings{}, err
	}
	cs, err := requireFloat(d, "CircleSize")
	if err != nil {
		return difficultySettings{}, err
	}
	od, err := requireFloat(d, "OverallDifficulty")
	if err != nil {
		return difficultySettings{}, err
	}

	ar := od
	if raw, ok := d["ApproachRate"]; ok {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			ar = parsed
		}
	}

	return difficultySettings{ar: ar, od: od, cs: cs, hp: hp}, nil
}

func requireFloat(d map[string]string, key string) (float64, error) {
	raw, ok := d[key]
	if !ok {
		return 0, fmt.Errorf("missing required [Difficulty] field %q", key)
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid [Difficulty] field %q: %w", key, err)
	}
	return v, nil
}

func parseFloatOr(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return v
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.Fields(raw)
}

func normalizeTimingPoints(points []osufile.RawTimingPoint) []domain.TimingPoint {
	out := make([]domain.TimingPoint, 0, len(points))
	for _, p := range points {
		bpm := 0.0
		if p.Uninherited && p.BeatLength > 0 {
			bpm = 60000.0 / p.BeatLength
		}
		out = append(out, domain.TimingPoint{
			Offset:      msToDuration(p.Offset),
			BeatLength:  p.BeatLength,
			Meter:       p.Meter,
			Uninherited: p.Uninherited,
			BPM:         bpm,
		})
	}
	return out
}

// timingState is the effective (beatLength, sliderVelocity) at a point in
// time, used to compute slider duration. An uninherited (red) line resets
// the slider velocity multiplier back to 1.0, matching osu!'s own rules.
type timingState struct {
	offset     float64
	beatLength float64
	velocity   float64 // slider velocity multiplier from the active inherited line
}

func buildTimingStates(points []osufile.RawTimingPoint) []timingState {
	sorted := make([]osufile.RawTimingPoint, len(points))
	copy(sorted, points)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Offset < sorted[j].Offset })

	states := make([]timingState, 0, len(sorted))
	beatLength := defaultBeatLengthMs
	velocity := 1.0
	for _, p := range sorted {
		if p.Uninherited {
			if p.BeatLength > 0 {
				beatLength = p.BeatLength
			}
			velocity = 1.0
		} else if p.BeatLength < 0 {
			velocity = -100.0 / p.BeatLength
		} else {
			velocity = 1.0
		}
		states = append(states, timingState{offset: p.Offset, beatLength: beatLength, velocity: velocity})
	}
	return states
}

func stateAt(states []timingState, t float64) timingState {
	if len(states) == 0 {
		return timingState{beatLength: defaultBeatLengthMs, velocity: 1.0}
	}
	idx := sort.Search(len(states), func(i int) bool { return states[i].offset > t })
	if idx == 0 {
		return timingState{offset: states[0].offset, beatLength: states[0].beatLength, velocity: 1.0}
	}
	return states[idx-1]
}

func normalizeHitObjects(raw []osufile.RawHitObject, rawTimingPoints []osufile.RawTimingPoint, sliderMultiplier float64) ([]domain.HitObject, int) {
	states := buildTimingStates(rawTimingPoints)

	out := make([]domain.HitObject, 0, len(raw))
	sliderCount := 0
	for _, h := range raw {
		switch h.Type {
		case osufile.RawHitObjectCircle:
			out = append(out, domain.HitObject{
				Type: domain.HitObjectCircle,
				X:    h.X, Y: h.Y,
				StartTime: msToDuration(h.Time),
				EndTime:   msToDuration(h.Time),
			})
		case osufile.RawHitObjectSlider:
			sliderCount++
			st := stateAt(states, h.Time)
			durationMs := sliderDurationMs(h.SliderLength, h.Slides, st.beatLength, st.velocity, sliderMultiplier)
			out = append(out, domain.HitObject{
				Type: domain.HitObjectSlider,
				X:    h.X, Y: h.Y,
				StartTime:       msToDuration(h.Time),
				EndTime:         msToDuration(h.Time + durationMs),
				Repeats:         h.Slides - 1,
				CurvePointCount: h.CurvePointCount,
			})
		case osufile.RawHitObjectSpinner:
			endTime := h.Time
			if h.EndTime >= h.Time {
				endTime = h.EndTime
			}
			out = append(out, domain.HitObject{
				Type: domain.HitObjectSpinner,
				X:    h.X, Y: h.Y,
				StartTime: msToDuration(h.Time),
				EndTime:   msToDuration(endTime),
			})
		default:
			// Mania hold notes and unrecognized types are dropped: this
			// pipeline normalizes std-style pools (docs/02-scope.md).
		}
	}
	return out, sliderCount
}

// sliderDurationMs implements osu!'s slider duration formula:
// duration = pixelLength * slides / (100 * sliderMultiplier * velocity) * beatLength
func sliderDurationMs(pixelLength float64, slides int, beatLength, velocity, sliderMultiplier float64) float64 {
	if pixelLength <= 0 || slides <= 0 || sliderMultiplier <= 0 {
		return 0
	}
	pixelsPerBeat := 100 * sliderMultiplier * velocity
	if pixelsPerBeat <= 0 {
		pixelsPerBeat = 100 * sliderMultiplier
	}
	beats := pixelLength * float64(slides) / pixelsPerBeat
	return beats * beatLength
}

// dominantBPM picks the uninherited BPM that covers the most playtime,
// since a single representative BPM is more useful to analyzers/reports
// than a list of every timing change.
func dominantBPM(points []osufile.RawTimingPoint, hitObjects []domain.HitObject) float64 {
	uninherited := make([]osufile.RawTimingPoint, 0, len(points))
	for _, p := range points {
		if p.Uninherited && p.BeatLength > 0 {
			uninherited = append(uninherited, p)
		}
	}
	if len(uninherited) == 0 {
		return 0
	}
	sort.Slice(uninherited, func(i, j int) bool { return uninherited[i].Offset < uninherited[j].Offset })

	mapEnd := uninherited[len(uninherited)-1].Offset
	if len(hitObjects) > 0 {
		if lastEnd := float64(hitObjects[len(hitObjects)-1].EndTime / time.Millisecond); lastEnd > mapEnd {
			mapEnd = lastEnd
		}
	}

	durationByBPM := map[float64]float64{}
	for i, p := range uninherited {
		segmentEnd := mapEnd
		if i+1 < len(uninherited) {
			segmentEnd = uninherited[i+1].Offset
		}
		duration := segmentEnd - p.Offset
		if duration < 0 {
			duration = 0
		}
		bpm := roundBPM(60000.0 / p.BeatLength)
		durationByBPM[bpm] += duration
	}

	bestBPM, bestDuration := 0.0, -1.0
	for bpm, duration := range durationByBPM {
		if duration > bestDuration {
			bestBPM, bestDuration = bpm, duration
		}
	}
	return bestBPM
}

func roundBPM(bpm float64) float64 {
	return float64(int(bpm*100+0.5)) / 100
}

func lengthSeconds(hitObjects []domain.HitObject) int {
	if len(hitObjects) == 0 {
		return 0
	}
	start := hitObjects[0].StartTime
	end := hitObjects[0].EndTime
	for _, h := range hitObjects {
		if h.StartTime < start {
			start = h.StartTime
		}
		if h.EndTime > end {
			end = h.EndTime
		}
	}
	return int((end - start).Round(time.Second).Seconds())
}

func msToDuration(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}

func hashSource(sourceBytes []byte) string {
	sum := sha256.Sum256(sourceBytes)
	return hex.EncodeToString(sum[:])
}
