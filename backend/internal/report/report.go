// Package report builds domain.Report values out of domain.Analysis
// results. It is a pure narration layer: Build never computes a new
// metric or invents a finding — it only selects, counts, and arranges
// data that an Analyzer already produced (docs/13-report-specification.md).
package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// Build assembles a domain.Report for scope out of analyses. Callers
// decide which Analyses belong in the report — e.g. every Analysis from
// one Engine.Run for a whole-tournament report, or a pre-filtered subtree
// for a narrower one. Build itself has no knowledge of the Tournament
// tree; it only narrates the Analyses it's given, keeping it testable
// Build constructs a report from analysis results for the given scope.
// It preserves existing findings, warnings, recommendations, and statistics, and sets the generation time from now().
func Build(scope domain.Scope, analyses []domain.Analysis, now func() time.Time) domain.Report {
	citations := citationsFrom(analyses)

	var warnings []domain.Citation
	for _, c := range citations {
		if c.Finding.Severity == domain.SeverityWarning || c.Finding.Severity == domain.SeverityCritical {
			warnings = append(warnings, c)
		}
	}

	stats := statistics(analyses, citations)

	return domain.Report{
		ScopeType:   scope.Type,
		ScopeID:     scope.ID,
		GeneratedAt: now(),
		Sections: domain.ReportSections{
			Summary:         summary(citations, stats),
			Findings:        citations,
			Warnings:        warnings,
			Recommendations: recommendations(citations),
			Statistics:      stats,
		},
	}
}

// citationsFrom flattens every Analysis's Findings into Citations, sorted
// by analyzer name then scope ID then severity (critical first) so a
// Report's Findings section has a stable, deterministic order regardless
// citationsFrom flattens analysis findings into a deterministically ordered list of citations.
// The returned citations are ordered by analyzer name, then scope ID, and then by severity with higher-severity findings first.
func citationsFrom(analyses []domain.Analysis) []domain.Citation {
	sorted := append([]domain.Analysis(nil), analyses...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].AnalyzerName != sorted[j].AnalyzerName {
			return sorted[i].AnalyzerName < sorted[j].AnalyzerName
		}
		return sorted[i].Scope.ID < sorted[j].Scope.ID
	})

	var citations []domain.Citation
	for _, a := range sorted {
		for _, f := range a.Findings {
			citations = append(citations, domain.Citation{
				AnalyzerName: a.AnalyzerName,
				Scope:        a.Scope,
				Finding:      f,
			})
		}
	}
	sort.SliceStable(citations, func(i, j int) bool {
		return severityRank(citations[i].Finding.Severity) > severityRank(citations[j].Finding.Severity)
	})
	return citations
}

// severityRank maps a severity to its sort order.
func severityRank(s domain.Severity) int {
	switch s {
	case domain.SeverityCritical:
		return 2
	case domain.SeverityWarning:
		return 1
	default:
		return 0
	}
}

// recommendations deduplicates Finding.Recommendation strings, preserving
// the order each was first cited in — later analyzers independently
// reaching the same recommendation (e.g. "diversify mapper selection")
// recommendations returns the unique non-empty recommendation strings from the citations in first-seen order.
func recommendations(citations []domain.Citation) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range citations {
		if c.Finding.Recommendation == "" || seen[c.Finding.Recommendation] {
			continue
		}
		seen[c.Finding.Recommendation] = true
		out = append(out, c.Finding.Recommendation)
	}
	return out
}

// statistics summarizes the Analyses and Citations with counts only —
// the raw-number content Architecture Principle 9 keeps out of Summary
// statistics computes count-based report metrics from analyses and citations.
// It includes total analyses, total findings, analyses with scores, the average
// score across scored analyses, and finding counts by severity.
func statistics(analyses []domain.Analysis, citations []domain.Citation) map[string]float64 {
	stats := map[string]float64{
		"total_analyses": float64(len(analyses)),
		"total_findings": float64(len(citations)),
	}

	var scoreSum float64
	var scoreCount int
	for _, a := range analyses {
		if a.Score != nil {
			scoreSum += *a.Score
			scoreCount++
		}
	}
	stats["analyses_with_score"] = float64(scoreCount)
	if scoreCount > 0 {
		stats["average_score"] = scoreSum / float64(scoreCount)
	}

	for _, c := range citations {
		switch c.Finding.Severity {
		case domain.SeverityInfo:
			stats["findings_info"]++
		case domain.SeverityWarning:
			stats["findings_warning"]++
		case domain.SeverityCritical:
			stats["findings_critical"]++
		}
	}

	return stats
}

// summary writes the narrative section: what happened and why it
// matters, built entirely out of Finding.Description text analyzers
// already wrote, never out of raw Statistics values
// summary builds the report narrative from citations and aggregate statistics.
// The statistics map must include the total number of analysis results under "total_analyses".
func summary(citations []domain.Citation, stats map[string]float64) string {
	if len(citations) == 0 {
		return fmt.Sprintf("No findings were raised across %d analysis result(s).", int(stats["total_analyses"]))
	}

	var critical, warning []domain.Citation
	for _, c := range citations {
		switch c.Finding.Severity {
		case domain.SeverityCritical:
			critical = append(critical, c)
		case domain.SeverityWarning:
			warning = append(warning, c)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "This report covers %d finding(s) across %d analysis result(s).", len(citations), int(stats["total_analyses"]))

	if len(critical) > 0 {
		b.WriteString(" ")
		fmt.Fprintf(&b, "%d critical issue(s) require attention: %s.", len(critical), describe(critical))
	}
	if len(warning) > 0 {
		b.WriteString(" ")
		fmt.Fprintf(&b, "%d warning(s) were also raised: %s.", len(warning), describe(warning))
	}
	if len(critical) == 0 && len(warning) == 0 {
		b.WriteString(" All raised findings are informational.")
	}

	return b.String()
}

// describe concatenates the finding descriptions from the given citations, separated by "; ".
func describe(citations []domain.Citation) string {
	descs := make([]string, len(citations))
	for i, c := range citations {
		descs[i] = c.Finding.Description
	}
	return strings.Join(descs, "; ")
}
