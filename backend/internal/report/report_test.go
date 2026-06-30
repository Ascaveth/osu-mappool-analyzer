package report

import (
	"testing"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

func fixedNow() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

func ptr(f float64) *float64 { return &f }

func TestBuild_NoFindingsProducesPositiveSummary(t *testing.T) {
	analyses := []domain.Analysis{
		{AnalyzerName: "composition-analyzer", Scope: domain.Scope{Type: domain.ScopeStage, ID: "stage-1"}},
	}

	r := Build(domain.Scope{Type: domain.ScopeTournament, ID: "t-1"}, analyses, fixedNow)

	if len(r.Sections.Findings) != 0 {
		t.Errorf("len(Findings) = %d, want 0", len(r.Sections.Findings))
	}
	if len(r.Sections.Warnings) != 0 {
		t.Errorf("len(Warnings) = %d, want 0", len(r.Sections.Warnings))
	}
	if r.Sections.Summary == "" {
		t.Error("Summary is empty, want a positive no-findings message")
	}
	if r.Sections.Statistics["total_analyses"] != 1 {
		t.Errorf("total_analyses = %v, want 1", r.Sections.Statistics["total_analyses"])
	}
}

func TestBuild_FindingsAreCitedAndCounted(t *testing.T) {
	analyses := []domain.Analysis{
		{
			AnalyzerName: "progression-analyzer",
			Scope:        domain.Scope{Type: domain.ScopeTournament, ID: "t-1"},
			Score:        ptr(0.5),
			Findings: []domain.Finding{
				{Severity: domain.SeverityWarning, Description: "OD drops between stages", Reason: "regression", Recommendation: "review beatmap selection"},
			},
		},
		{
			AnalyzerName: "composition-analyzer",
			Scope:        domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
			Findings: []domain.Finding{
				{Severity: domain.SeverityCritical, Description: "one mapper supplies the entire stage", Reason: "single style tested", Recommendation: "diversify mapper selection"},
			},
		},
	}

	r := Build(domain.Scope{Type: domain.ScopeTournament, ID: "t-1"}, analyses, fixedNow)

	if len(r.Sections.Findings) != 2 {
		t.Fatalf("len(Findings) = %d, want 2", len(r.Sections.Findings))
	}
	// Critical findings are ranked before warnings.
	if r.Sections.Findings[0].Finding.Severity != domain.SeverityCritical {
		t.Errorf("Findings[0].Severity = %v, want critical (sorted first)", r.Sections.Findings[0].Finding.Severity)
	}
	if len(r.Sections.Warnings) != 2 {
		t.Errorf("len(Warnings) = %d, want 2 (warning + critical both count)", len(r.Sections.Warnings))
	}
	if len(r.Sections.Recommendations) != 2 {
		t.Errorf("len(Recommendations) = %d, want 2", len(r.Sections.Recommendations))
	}
	if r.Sections.Statistics["findings_warning"] != 1 {
		t.Errorf("findings_warning = %v, want 1", r.Sections.Statistics["findings_warning"])
	}
	if r.Sections.Statistics["findings_critical"] != 1 {
		t.Errorf("findings_critical = %v, want 1", r.Sections.Statistics["findings_critical"])
	}
	if r.Sections.Statistics["average_score"] != 0.5 {
		t.Errorf("average_score = %v, want 0.5", r.Sections.Statistics["average_score"])
	}
	if r.Sections.Summary == "" {
		t.Error("Summary is empty")
	}
}

func TestBuild_DeduplicatesRecommendations(t *testing.T) {
	analyses := []domain.Analysis{
		{
			AnalyzerName: "a",
			Scope:        domain.Scope{Type: domain.ScopeStage, ID: "stage-1"},
			Findings: []domain.Finding{
				{Severity: domain.SeverityWarning, Description: "d1", Reason: "r1", Recommendation: "same fix"},
			},
		},
		{
			AnalyzerName: "b",
			Scope:        domain.Scope{Type: domain.ScopeStage, ID: "stage-2"},
			Findings: []domain.Finding{
				{Severity: domain.SeverityWarning, Description: "d2", Reason: "r2", Recommendation: "same fix"},
			},
		},
	}

	r := Build(domain.Scope{Type: domain.ScopeTournament, ID: "t-1"}, analyses, fixedNow)

	if len(r.Sections.Recommendations) != 1 {
		t.Fatalf("len(Recommendations) = %d, want 1 (deduplicated)", len(r.Sections.Recommendations))
	}
}

func TestBuild_SetsScopeAndTimestamp(t *testing.T) {
	r := Build(domain.Scope{Type: domain.ScopeStage, ID: "stage-1"}, nil, fixedNow)

	if r.ScopeType != domain.ScopeStage || r.ScopeID != "stage-1" {
		t.Errorf("Scope = %v/%v, want stage/stage-1", r.ScopeType, r.ScopeID)
	}
	if !r.GeneratedAt.Equal(fixedNow()) {
		t.Errorf("GeneratedAt = %v, want %v", r.GeneratedAt, fixedNow())
	}
}
