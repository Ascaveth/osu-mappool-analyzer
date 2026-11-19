package sheetimport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

const MaxRows = 256

var sheetPath = regexp.MustCompile(`^/spreadsheets/d/([a-zA-Z0-9_-]{1,128})/edit/?$`)
var digits = regexp.MustCompile(`^[0-9]+$`)

func ExportURL(link string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(link))
	invalid := errors.New("Paste a Google Sheets tab link that includes a gid.")
	if err != nil || u.Scheme != "https" || u.Host != "docs.google.com" || u.User != nil {
		return "", invalid
	}
	match := sheetPath.FindStringSubmatch(u.Path)
	gid := u.Query().Get("gid")
	if gid == "" {
		fragment, _ := url.ParseQuery(u.Fragment)
		gid = fragment.Get("gid")
	}
	if len(match) == 0 || len(gid) > 10 || !digits.MatchString(gid) {
		return "", invalid
	}
	return "https://docs.google.com/spreadsheets/d/" + match[1] + "/export?format=csv&gid=" + gid, nil
}

type Assignment struct {
	Row        int    `json:"row"`
	SlotID     string `json:"slot_id"`
	MapID      int64  `json:"map_id"`
	PreviousID string `json:"previous_id"`
	Checksum   string `json:"checksum"`
}

type Row struct {
	Assignment
	Stage     string `json:"stage"`
	Slot      string `json:"slot"`
	Title     string `json:"title"`
	Previous  string `json:"previous,omitempty"`
	Unchanged bool   `json:"unchanged"`
	Error     string `json:"error,omitempty"`
	Warning   string `json:"warning,omitempty"`
}

func Parse(source io.Reader) ([]Row, error) {
	reader := csv.NewReader(source)
	reader.FieldsPerRecord = -1
	rows := []Row{}
	var previous []string
	var stage string
	modCol, mapCol, titleCol := -1, -1, -1
	foundHeader := false
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("We couldn't read this tab as CSV. Check its layout and try again.")
		}
		line, _ := reader.FieldPos(0)
		mod, mapID, title := column(record, "MOD#"), column(record, "Map ID"), column(record, "Artist - Title [Diff]")
		if (mod >= 0) != (mapID >= 0) {
			return nil, fmt.Errorf("The table header on row %d needs both MOD# and Map ID columns.", line)
		}
		if mod >= 0 && mapID >= 0 {
			// The stage heading sits immediately to the left of MOD# on the preceding row.
			stage = cell(previous, mod-1)
			modCol, mapCol, titleCol = mod, mapID, title
			foundHeader = true
		} else if modCol >= 0 {
			slot, id, title := cell(record, modCol), cell(record, mapCol), cell(record, titleCol)
			if slot != "" || id != "" || title != "" {
				row := Row{Assignment: Assignment{Row: line}, Stage: stage, Slot: slot, Title: title}
				row.MapID, _ = strconv.ParseInt(id, 10, 64)
				switch {
				case stage == "":
					row.Error = "Add a stage heading above this block."
				case slot == "":
					row.Error = "Enter a slot label in the MOD# column."
				case !digits.MatchString(id) || row.MapID <= 0 || row.MapID > 2147483647:
					row.Error = "Enter a valid numeric osu! beatmap ID in the Map ID column."
				}
				rows = append(rows, row)
				if len(rows) > MaxRows {
					return nil, fmt.Errorf("This tab has more than %d map rows. Split it into smaller tabs and try again.", MaxRows)
				}
			}
		}
		previous = record
	}
	if !foundHeader {
		return nil, errors.New("This tab uses an unsupported layout. Each stage needs a heading and a table with MOD# and Map ID columns.")
	}
	return rows, nil
}

func column(record []string, label string) int {
	for i, value := range record {
		if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(value, "\uFEFF")), label) {
			return i
		}
	}
	return -1
}

func cell(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func MapRows(rows []Row, tournament *domain.Tournament) {
	targets := make(map[string][]int)
	maps := make(map[int64][]int)
	for i := range rows {
		row := &rows[i]
		if row.Error != "" {
			continue
		}
		mapRow(row, tournament)
		if row.SlotID != "" {
			targets[row.SlotID] = append(targets[row.SlotID], i)
			maps[row.MapID] = append(maps[row.MapID], i)
		}
	}
	for _, indices := range targets {
		if len(indices) > 1 {
			for _, i := range indices {
				rows[i].Error = fmt.Sprintf("Rows %s target the same slot. Give each row a different slot.", rowNumbers(rows, indices))
			}
		}
	}
	for _, indices := range maps {
		var locations []string
		for _, i := range indices {
			if rows[i].Error == "" {
				locations = append(locations, fmt.Sprintf("row %d (%s / %s)", rows[i].Row, rows[i].Stage, rows[i].Slot))
			}
		}
		if len(locations) < 2 {
			continue
		}
		for _, i := range indices {
			if rows[i].Error == "" {
				rows[i].Warning = fmt.Sprintf("The same beatmap appears in %s. Check that this is intentional.", strings.Join(locations, ", "))
			}
		}
	}
}

func rowNumbers(rows []Row, indices []int) string {
	numbers := make([]string, 0, len(indices))
	for _, i := range indices {
		numbers = append(numbers, strconv.Itoa(rows[i].Row))
	}
	return strings.Join(numbers, ", ")
}

func mapRow(row *Row, tournament *domain.Tournament) {
	var matches []domain.Stage
	var names []string
	for _, stage := range tournament.Stages {
		names = append(names, stage.Name)
		if strings.EqualFold(strings.TrimSpace(stage.Name), row.Stage) {
			matches = append(matches, stage)
		}
	}
	if len(matches) != 1 {
		row.Error = fmt.Sprintf("The stage %q must match exactly one stage in this pool. Available stages: %s.", row.Stage, strings.Join(names, ", "))
		return
	}

	var slots []domain.Slot
	var labels []string
	for _, category := range matches[0].Categories {
		name := strings.TrimSpace(category.Name)
		for _, slot := range category.Slots {
			label := name + strconv.Itoa(slot.Position)
			if strings.EqualFold(row.Slot, label) || (strings.EqualFold(row.Slot, "FM") && strings.EqualFold(name, "FM")) {
				slots = append(slots, slot)
				labels = append(labels, label)
			}
		}
	}
	switch len(slots) {
	case 0:
		row.Error = fmt.Sprintf("Create or correct slot %s in %s, then review the import again.", row.Slot, matches[0].Name)
	case 1:
		slot := slots[0]
		row.Stage, row.Slot = matches[0].Name, labels[0]
		row.SlotID = slot.ID
		row.PreviousID = storage.SlotBeatmapID(&slot)
		if slot.Beatmap != nil {
			row.Previous = beatmapTitle(slot.Beatmap)
			row.Unchanged = slot.Beatmap.OsuBeatmapID != nil && *slot.Beatmap.OsuBeatmapID == row.MapID
		}
	default:
		row.Error = fmt.Sprintf("Slot %s in %s is ambiguous. Enter the intended label in MOD#: %s.", row.Slot, matches[0].Name, strings.Join(labels, ", "))
	}
}

func beatmapTitle(beatmap *domain.Beatmap) string {
	return fmt.Sprintf("%s - %s [%s]", beatmap.Artist, beatmap.Title, beatmap.Version)
}
