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
// Beatmap builds a domain beatmap from raw parsed beatmap data and the original source bytes.
// It preserves raw metadata, normalizes difficulty, timing points, and hit objects, and computes
// derived values such as BPM, length, slider ratio, and source-file hash. Returns an error if raw is nil.
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
		OsuBeatmapID: parseOsuBeatmapID(raw.Metadata["BeatmapID"]),
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
// Otherwise, it sets AR to the overall difficulty value.
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
		parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return difficultySettings{}, fmt.Errorf("invalid [Difficulty] field %q: %w", "ApproachRate", err)
		}
		ar = parsed
	}

	return difficultySettings{ar: ar, od: od, cs: cs, hp: hp}, nil
}

// requireFloat returns the parsed value of a required [Difficulty] field.
// It reports an error if the field is missing or cannot be parsed as a float.
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

// parseOsuBeatmapID parses the .osu file's [Metadata] BeatmapID field into
// osu!'s numeric beatmap ID. Returns nil when the field is absent, blank,
// non-numeric, or non-positive (0/negative BeatmapID values appear in
// locally-authored/never-submitted .osu files and don't identify a real
// osu! beatmap) — a nil OsuBeatmapID is a permanent, expected state for
// such maps, resolved later at enrichment time via checksum lookup
// (internal/enrich), not an error here.
func parseOsuBeatmapID(raw string) *int64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

// parseFloatOr parses raw as a float64 and returns fallback if parsing fails.
func parseFloatOr(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return v
}

// splitTags splits a tags string into whitespace-separated fields.
// It returns nil when the input is blank or contains only whitespace.
func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.Fields(raw)
}

// normalizeTimingPoints converts raw timing points into domain timing points and
// sets BPM for uninherited points with a positive beat length.
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

// buildTimingStates creates a sorted timeline of effective beat length and slider velocity changes from timing points.
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

// stateAt returns the effective timing state at time t.
// If no timing states are available, it returns the default beat length and
// a velocity of 1.0. If t falls before the first state, it returns the first
// state's beat length with a velocity of 1.0.
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

// normalizeHitObjects converts raw hit objects into domain hit objects and counts sliders.
// It preserves circles and spinners, derives slider end times from timing data, and drops unsupported objects.
// It returns the normalized hit objects and the number of sliders encountered.
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
// sliderDurationMs computes the duration of an osu! slider in milliseconds.
// It uses the slider's pixel length, repeat count, active beat length, inherited
// velocity, and map slider multiplier.
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
// dominantBPM returns the BPM whose uninherited timing segments cover the most playtime.
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
	for _, ho := range hitObjects {
		if endMs := float64(ho.EndTime / time.Millisecond); endMs > mapEnd {
			mapEnd = endMs
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

// roundBPM rounds a BPM value to two decimal places.
func roundBPM(bpm float64) float64 {
	return float64(int(bpm*100+0.5)) / 100
}

// lengthSeconds computes the duration covered by the hit objects, rounded to the nearest second.
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

// msToDuration converts a millisecond value to a time.Duration.
func msToDuration(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}

// hashSource returns the SHA-256 hash of sourceBytes as a lowercase hexadecimal string.
func hashSource(sourceBytes []byte) string {
	sum := sha256.Sum256(sourceBytes)
	return hex.EncodeToString(sum[:])
}
