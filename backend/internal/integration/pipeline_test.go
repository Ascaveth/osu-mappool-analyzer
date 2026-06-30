// Package integration proves the full pipeline documented in
// docs/04-architecture-principles.md (Raw Data -> Normalization ->
// Analyzer -> Report) actually composes end to end: real .osu fixtures
// parsed, normalized into domain.Beatmap values, slotted into a small
// domain.Tournament, run through the registered Tournament Analyzers via
// the Engine, and narrated into a domain.Report. No existing test
// exercises every stage in one run — each package's own tests cover its
// stage in isolation against synthetic data.
package integration

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/tournament"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/normalize"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osufile"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/report"
)

func loadBeatmap(t *testing.T, id, path string) *domain.Beatmap {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	raw, err := osufile.Parse(bytes.NewReader(source))
	if err != nil {
		t.Fatalf("osufile.Parse(%s): %v", path, err)
	}
	bm, err := normalize.Beatmap(raw, source)
	if err != nil {
		t.Fatalf("normalize.Beatmap(%s): %v", path, err)
	}
	bm.ID = id
	return bm
}

// buildTournament wires real, parsed-and-normalized beatmaps into a small
// two-stage Tournament. Both stages reuse the same fixture beatmap files
// so the resulting pool is intentionally low-diversity and lopsided
// (one category holding every slot) — this is meant to give the
// tournament analyzers something to actually find, proving findings flow
// all the way through to the Report, not just that the pipeline runs
// without error on a clean pool.
func buildTournament(t *testing.T) *domain.Tournament {
	t.Helper()

	qualNM := loadBeatmap(t, "bm-qual-nm", "../osufile/testdata/sample.osu")
	finalsNM := loadBeatmap(t, "bm-finals-nm", "../osufile/testdata/extreme_values.osu")

	return &domain.Tournament{
		ID: "t-integration", Name: "Integration Test Open", Edition: "2026",
		Stages: []domain.Stage{
			{
				ID: "stage-qualifiers", Name: "Qualifiers", Order: 1,
				Categories: []domain.Category{
					{
						ID: "cat-qual-nm", Name: "NM", Order: 1,
						Slots: []domain.Slot{
							{ID: "slot-1", Position: 1, Beatmap: qualNM},
							{ID: "slot-2", Position: 2, Beatmap: qualNM}, // same beatmap reused: low diversity, on purpose
						},
					},
				},
			},
			{
				ID: "stage-finals", Name: "Finals", Order: 2,
				Categories: []domain.Category{
					{
						ID: "cat-finals-nm", Name: "NM", Order: 1,
						Slots: []domain.Slot{
							{ID: "slot-3", Position: 1, Beatmap: finalsNM},
						},
					},
				},
			},
		},
	}
}

func buildEngine(t *testing.T) *analysis.Engine {
	t.Helper()
	e := analysis.NewEngine()
	e.Now = func() time.Time { return time.Unix(0, 0).UTC() }

	for _, a := range []analysis.Analyzer{
		tournament.CompositionAnalyzer{},
		tournament.ProgressionAnalyzer{},
		tournament.BalanceAnalyzer{},
		tournament.DiversityAnalyzer{},
	} {
		if err := e.Register(a); err != nil {
			t.Fatalf("Register(%s): %v", a.Name(), err)
		}
	}
	return e
}

func TestPipeline_ParseNormalizeAnalyzeReport(t *testing.T) {
	tour := buildTournament(t)

	if issues := domain.ValidateConfiguration(tour); domain.HasErrors(issues) {
		t.Fatalf("test fixture tournament configuration should be valid, got issues: %+v", issues)
	}

	engine := buildEngine(t)
	analyses, err := engine.Run(context.Background(), tour)
	if err != nil {
		t.Fatalf("Engine.Run returned error: %v", err)
	}
	if len(analyses) == 0 {
		t.Fatal("Engine.Run produced no analyses")
	}

	rep := report.Build(domain.Scope{Type: domain.ScopeTournament, ID: tour.ID}, analyses, func() time.Time { return time.Unix(0, 0).UTC() })

	if rep.Sections.Summary == "" {
		t.Error("Report.Sections.Summary should not be empty")
	}
	if got := rep.Sections.Statistics["total_analyses"]; int(got) != len(analyses) {
		t.Errorf("Statistics[total_analyses] = %v, want %d", got, len(analyses))
	}

	// The fixture deliberately reuses one beatmap across two slots in the
	// same category (low song diversity) — diversity-analyzer should
	// surface a finding about it, proving a real Finding produced deep in
	// the analyzer made it all the way to the Report's citations.
	foundDiversityCitation := false
	for _, c := range rep.Sections.Findings {
		if c.AnalyzerName == "diversity-analyzer" {
			foundDiversityCitation = true
		}
		if c.Finding.Severity == "" || c.Finding.Reason == "" || c.Finding.Recommendation == "" {
			t.Errorf("citation from %q has an incomplete finding: %+v", c.AnalyzerName, c.Finding)
		}
	}
	if !foundDiversityCitation {
		t.Error("expected at least one diversity-analyzer citation given the deliberately reused beatmap")
	}
}

func TestPipeline_InvalidTournamentConfigurationIsCaughtBeforeAnalysis(t *testing.T) {
	tour := buildTournament(t)
	tour.Stages[0].Categories[0].Order = 1
	extra := tour.Stages[0].Categories[0]
	extra.ID = "cat-qual-dupe"
	extra.Name = "HD"
	extra.Order = 1 // collides with cat-qual-nm's order within the same stage
	tour.Stages[0].Categories = append(tour.Stages[0].Categories, extra)

	issues := domain.ValidateConfiguration(tour)
	if !domain.HasErrors(issues) {
		t.Fatal("expected a hard error for duplicate category order within a stage")
	}
}
