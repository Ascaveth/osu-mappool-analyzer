package osufile

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ErrNotAnOsuFile is returned when the input doesn't start with a
// recognizable "osu file format vNN" header line.
var ErrNotAnOsuFile = fmt.Errorf("osufile: input is not a valid .osu file")

type section int

const (
	sectionNone section = iota
	sectionGeneral
	sectionMetadata
	sectionDifficulty
	sectionTimingPoints
	sectionHitObjects
	sectionOther
)

// Parse reads a .osu file and returns its raw, format-faithful contents.
// It returns ErrNotAnOsuFile if the input does not look like a .osu file,
// and otherwise tolerates malformed individual lines by skipping them —
// a single bad hit object line should not fail an entire import.
func Parse(r io.Reader) (*RawBeatmap, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	raw := &RawBeatmap{
		General:    map[string]string{},
		Metadata:   map[string]string{},
		Difficulty: map[string]string{},
	}

	if !scanner.Scan() {
		return nil, ErrNotAnOsuFile
	}
	header := strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "\uFEFF")
	version, ok := parseFormatVersion(header)
	if !ok {
		return nil, ErrNotAnOsuFile
	}
	raw.FormatVersion = version

	current := sectionNone
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = sectionFromHeader(line)
			continue
		}

		switch current {
		case sectionGeneral:
			parseKeyValue(line, raw.General)
		case sectionMetadata:
			parseKeyValue(line, raw.Metadata)
		case sectionDifficulty:
			parseKeyValue(line, raw.Difficulty)
		case sectionTimingPoints:
			if tp, ok := parseTimingPoint(line); ok {
				raw.TimingPoints = append(raw.TimingPoints, tp)
			}
		case sectionHitObjects:
			if ho, ok := parseHitObject(line); ok {
				raw.HitObjects = append(raw.HitObjects, ho)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("osufile: reading input: %w", err)
	}

	return raw, nil
}

func parseFormatVersion(header string) (int, bool) {
	const prefix = "osu file format v"
	if !strings.HasPrefix(header, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return 0, false
	}
	return n, true
}

func sectionFromHeader(header string) section {
	switch strings.Trim(header, "[]") {
	case "General":
		return sectionGeneral
	case "Metadata":
		return sectionMetadata
	case "Difficulty":
		return sectionDifficulty
	case "TimingPoints":
		return sectionTimingPoints
	case "HitObjects":
		return sectionHitObjects
	default:
		return sectionOther
	}
}

// parseKeyValue parses "Key: Value" lines used by [General]/[Metadata]/[Difficulty].
func parseKeyValue(line string, dst map[string]string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return
	}
	dst[key] = value
}

// parseTimingPoint parses one comma-separated [TimingPoints] line:
// offset,beatLength,meter,sampleSet,sampleIndex,volume,uninherited,effects
func parseTimingPoint(line string) (RawTimingPoint, bool) {
	fields := strings.Split(line, ",")
	if len(fields) < 2 {
		return RawTimingPoint{}, false
	}

	offset, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
	if err != nil {
		return RawTimingPoint{}, false
	}
	beatLength, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
	if err != nil {
		return RawTimingPoint{}, false
	}

	meter := 4
	if len(fields) > 2 {
		if m, err := strconv.Atoi(strings.TrimSpace(fields[2])); err == nil {
			meter = m
		}
	}

	uninherited := true
	if len(fields) > 6 {
		uninherited = strings.TrimSpace(fields[6]) != "0"
	}

	return RawTimingPoint{
		Offset:      offset,
		BeatLength:  beatLength,
		Meter:       meter,
		Uninherited: uninherited,
	}, true
}

// parseHitObject parses one comma-separated [HitObjects] line:
// x,y,time,type,hitSound,objectParams...,hitSample
func parseHitObject(line string) (RawHitObject, bool) {
	fields := strings.Split(line, ",")
	if len(fields) < 4 {
		return RawHitObject{}, false
	}

	x, err := strconv.Atoi(strings.TrimSpace(fields[0]))
	if err != nil {
		return RawHitObject{}, false
	}
	y, err := strconv.Atoi(strings.TrimSpace(fields[1]))
	if err != nil {
		return RawHitObject{}, false
	}
	t, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
	if err != nil {
		return RawHitObject{}, false
	}
	typeBits, err := strconv.Atoi(strings.TrimSpace(fields[3]))
	if err != nil {
		return RawHitObject{}, false
	}

	ho := RawHitObject{
		X: x, Y: y, Time: t,
		SliderLength: -1,
		EndTime:      -1,
	}

	switch {
	case typeBits&8 != 0: // spinner: x,y,time,type,hitSound,endTime,hitSample
		ho.Type = RawHitObjectSpinner
		if len(fields) > 5 {
			if et, err := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64); err == nil {
				ho.EndTime = et
			}
		}
	case typeBits&2 != 0: // slider: x,y,time,type,hitSound,curveType|curvePoints,slides,length,...
		ho.Type = RawHitObjectSlider
		ho.Slides = 1
		if len(fields) > 5 {
			curve := strings.Split(strings.TrimSpace(fields[5]), "|")
			if len(curve) > 0 {
				ho.CurveType = curve[0]
				ho.CurvePointCount = len(curve) - 1
			}
		}
		if len(fields) > 6 {
			if s, err := strconv.Atoi(strings.TrimSpace(fields[6])); err == nil && s > 0 {
				ho.Slides = s
			}
		}
		if len(fields) > 7 {
			if l, err := strconv.ParseFloat(strings.TrimSpace(fields[7]), 64); err == nil {
				ho.SliderLength = l
			}
		}
	case typeBits&1 != 0: // circle: x,y,time,type,hitSound,hitSample
		ho.Type = RawHitObjectCircle
	default:
		// Mania hold notes (bit 128) and unrecognized types: dropped by
		// the caller during normalization, not by the parser, so callers
		// can decide whether to count/report skipped objects.
		ho.Type = RawHitObjectUnknown
	}

	return ho, true
}
