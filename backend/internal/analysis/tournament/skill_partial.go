package tournament

import "github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"

func partialSkillFinding() domain.Finding {
	return domain.Finding{Severity: domain.SeverityWarning, Description: "mechanical skill analysis is partial: some placements have no resolved valid fixed mods", Reason: "FM/TB require tournament regulations; unknown categories and conflicting mods cannot define a fixed playing profile", Recommendation: "interpret results only for analyzed placements; HD/FL visibility demands are not modeled"}
}

func partialSkillFindings(skipped int) []domain.Finding {
	if skipped == 0 {
		return nil
	}
	return []domain.Finding{partialSkillFinding()}
}
