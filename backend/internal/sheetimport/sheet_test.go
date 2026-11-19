package sheetimport

import (
	"os"
	"strings"
	"testing"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func TestParseStageBlocks(t *testing.T) {
	source, err := os.Open("testdata/stage-blocks.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	rows, err := Parse(source)
	if err != nil || len(rows) != 11 {
		t.Fatalf("Parse rows = %d, error = %v", len(rows), err)
	}
	for i, row := range rows {
		if row.Stage != "Qualifiers" || row.Row != 175+i || row.Error != "" {
			t.Errorf("unexpected row: %+v", row)
		}
	}
	if rows[0].MapID != 4134924 || rows[10].MapID != 3093177 || rows[10].Slot != "FM" {
		t.Fatalf("unexpected first or last row: %+v, %+v", rows[0], rows[10])
	}
}

func TestExportURL(t *testing.T) {
	got, err := ExportURL("https://docs.google.com/spreadsheets/d/example-id/edit#gid=0")
	if err != nil || got != "https://docs.google.com/spreadsheets/d/example-id/export?format=csv&gid=0" {
		t.Fatalf("%q, %v", got, err)
	}
	for _, link := range []string{
		"https://other.example/spreadsheets/d/id/edit?gid=0",
		"https://docs.google.com/spreadsheets/d/id/edit",
		"https://docs.google.com/spreadsheets/d/id/edit?gid=x",
	} {
		if _, err := ExportURL(link); err == nil {
			t.Errorf("accepted %q", link)
		}
	}
}

func TestMapRows(t *testing.T) {
	rows, err := Parse(strings.NewReader(",Qualifiers,,\n,,MOD#,Map ID\n,,NM1,123\n,,FM,456\n,,HD1,789\n,,HR1,invalid\n,,NM2,432\n,,NM2,543\n,Finals,,\n,,MOD#,Map ID\n,,NM1,123\n"))
	if err != nil {
		t.Fatal(err)
	}
	tournament := &domain.Tournament{Stages: []domain.Stage{
		{Name: " qualifiers ", Categories: []domain.Category{
			{Name: "NM", Slots: []domain.Slot{{ID: "q1", Position: 1}, {ID: "q2", Position: 2}}},
			{Name: "FM", Slots: []domain.Slot{{ID: "fm1", Position: 1}, {ID: "fm2", Position: 2}}},
		}},
		{Name: "Finals", Categories: []domain.Category{{Name: "NM", Slots: []domain.Slot{{ID: "f1", Position: 1}}}}},
	}}
	MapRows(rows, tournament)
	if rows[0].SlotID != "q1" || rows[6].SlotID != "f1" || rows[0].Warning == "" || rows[6].Warning == "" {
		t.Fatalf("stage or repeated-map mapping incorrect: %+v", rows)
	}
	for i, want := range map[int]string{1: "ambiguous", 2: "Create or correct", 3: "valid numeric", 4: "same slot", 5: "same slot"} {
		if !strings.Contains(rows[i].Error, want) {
			t.Errorf("row %d: error = %q", i, rows[i].Error)
		}
	}
	tournament.Stages[0].Categories[1].Slots = tournament.Stages[0].Categories[1].Slots[:1]
	fm := []Row{{Stage: "Qualifiers", Slot: "FM"}}
	MapRows(fm, tournament)
	if fm[0].SlotID != "fm1" || fm[0].Error != "" {
		t.Fatalf("single FM: %+v", fm[0])
	}
}

func TestParseReportsUnsupportedHeaders(t *testing.T) {
	for _, input := range []string{
		",Qualifiers,,\n,,Slot,ID\n,,NM1,123\n",
		",Qualifiers,,\n,,MOD#,ID\n,,NM1,123\n",
	} {
		if _, err := Parse(strings.NewReader(input)); err == nil {
			t.Errorf("accepted unsupported header in %q", input)
		}
	}
}

func TestParseReportsIncompleteRows(t *testing.T) {
	rows, err := Parse(strings.NewReader(",Qualifiers,,,\n,,MOD#,Artist - Title [Diff],Map ID\n,,,Missing slot,123\n,,HD1,Missing ID,\n"))
	if err != nil || len(rows) != 2 {
		t.Fatalf("Parse rows = %d, error = %v", len(rows), err)
	}
	if rows[0].Row != 3 || !strings.Contains(rows[0].Error, "slot label") ||
		rows[1].Row != 4 || !strings.Contains(rows[1].Error, "beatmap ID") {
		t.Fatalf("incomplete rows: %+v", rows)
	}
}
