package tournament

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/pattern"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// Skillset is a tournament-relevant tag describing a mechanical skill a
// beatmap primarily tests (aim, stream, tech, jump, alt/finger-control,
// etc.). It is intentionally free text, like domain.Category.Name — never
// validated against a fixed enum, since what "skills" a tournament cares
// about is itself part of its user-defined structure
// (docs/04-architecture-principles.md, Principle 4).
type Skillset string

const skillsetUnclassified Skillset = "unclassified"

// SkillsetRule is one classification rule: a beatmap whose SkillsetProfile
// satisfies Match is tagged with Skillset. A beatmap can satisfy more than
// one rule — mixed-skillset beatmaps are expected, not an edge case to
// eliminate (see Analyze's doc comment on filled-slot vs. tag-instance
// counting).
type SkillsetRule struct {
	Skillset Skillset
	Match    func(pattern.SkillsetProfile) bool
}

// Thresholds behind DefaultTaxonomy. Named and documented individually so
// they can be revisited independently, following the same convention
// pattern.StreamBurstAnalyzer's streamSnapDivisor/burstMinLength/
// streamMinLength already establish: these are stated conventions for
// explainability, not measured facts about what makes a beatmap "aim" or
// "tech".
const (
	// jumpDistanceThreshold (osu!pixels) marks a beatmap's average jump
	// distance as wide enough to call the map "jump"/"aim"-oriented.
	jumpDistanceThreshold = 150.0

	// techAnchorCountThreshold marks a beatmap's average slider anchor
	// count as complex enough to call the map "tech"-oriented.
	techAnchorCountThreshold = 4.0

	// lowReverseSliderRatio marks a beatmap as slider-simple enough (few
	// reverse sliders) that wide jump spacing reads as "aim" rather than
	// "tech" — a beatmap can be both wide-spaced and slider-complex, in
	// which case it is tagged tech, not aim (tech's Match is checked
	// independently, so this only narrows aim's Match).
	lowReverseSliderRatio = 0.2

	// altMinRunLength marks a dense-but-not-stream run of circles (below
	// pattern's own stream threshold) as "alt"/finger-control-oriented.
	altMinRunLength = 3
)

// DefaultTaxonomy returns the built-in skillset classification rules.
// Every threshold here is a named constant precisely so it can be
// overridden or extended by supplying a custom Taxonomy to
// SkillCoverageAnalyzer, rather than by editing this function — adding or
// changing a tournament's skillset definitions must never require
// modifying analyzer code (docs/04-architecture-principles.md, Principle
// 4). Full per-tournament user-editable taxonomies (a
// domain.Tournament-level field, configurable through the API/UI) are a
// deferred follow-up: today, no per-tournament settings surface exists in
// the domain model at all, and shipping a named default first — like
// pattern.StreamBurstAnalyzer's snap conventions — is this codebase's
// established precedent over building configurability speculatively.
func DefaultTaxonomy() []SkillsetRule {
	return []SkillsetRule{
		{Skillset: "stream", Match: func(p pattern.SkillsetProfile) bool {
			return p.StreamCount > 0
		}},
		{Skillset: "tech", Match: func(p pattern.SkillsetProfile) bool {
			return p.AvgAnchorCount >= techAnchorCountThreshold
		}},
		{Skillset: "alt", Match: func(p pattern.SkillsetProfile) bool {
			return p.StreamCount == 0 && p.LongestRunLength >= altMinRunLength
		}},
		{Skillset: "aim", Match: func(p pattern.SkillsetProfile) bool {
			return p.AvgJumpDistance >= jumpDistanceThreshold && p.ReverseSliderRatio < lowReverseSliderRatio
		}},
		{Skillset: "jump", Match: func(p pattern.SkillsetProfile) bool {
			return p.AvgJumpDistance >= jumpDistanceThreshold
		}},
	}
}

// skillsetMajorityThreshold marks a stage as skillset-overloaded when one
// skillset's share of filled slots exceeds it. Set higher than
// categoryMajorityThreshold (0.5) because skillset tags don't partition
// filled slots the way categories do — a beatmap can count toward more
// than one skillset, so "share" here is looser than a strict majority.
const skillsetMajorityThreshold = 0.6

// minSlotsForCoverageJudgment is the fewest filled slots a stage needs
// before "this taxonomy skillset has zero representation" is a meaningful
// finding rather than noise from a stage that simply doesn't have enough
// maps yet to cover a multi-item taxonomy (mirrors BalanceAnalyzer's
// len(values) > 1 guard and ProgressionAnalyzer's len(sequence) < 2 guard).
const minSlotsForCoverageJudgment = 3

// SkillCoverageAnalyzer reports, per Stage, how filled slots are
// distributed across a tournament-relevant skillset taxonomy (aim, stream,
// tech, jump, alt, etc.) — the stage-level counterpart to BalanceAnalyzer's
// numeric-variance check (AR/OD/slider ratio) and CompositionAnalyzer's
// category/mapper share check, but for mechanical skill coverage rather
// than difficulty settings or metadata.
//
// This closes the "missing skill coverage" gap named in
// docs/12-tournament-analyzers.md and docs/03-terminology.md: real-world
// tournament mappool feedback consistently identifies skillset imbalance
// (a pool over-indexing on one skillset, e.g. all tech or all aim) as the
// dominant cause of "unbalanced"/"not fun" complaints — more so than raw
// difficulty-number variance, which BalanceAnalyzer already covers.
type SkillCoverageAnalyzer struct {
	// Taxonomy is the ordered list of skillset classification rules this
	// analyzer evaluates. Nil means DefaultTaxonomy().
	Taxonomy []SkillsetRule
}

func (SkillCoverageAnalyzer) Name() string { return "skill-coverage-analyzer" }

func (SkillCoverageAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeStage }

func (a SkillCoverageAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	stage := analysis.FindStage(in.Tournament, in.Scope.ID)
	if stage == nil {
		return analysis.Result{}, fmt.Errorf("tournament: stage %q not found in tournament", in.Scope.ID)
	}

	taxonomy := a.Taxonomy
	if taxonomy == nil {
		taxonomy = DefaultTaxonomy()
	}

	filledSlots := 0
	// skillsetCounts can sum to more than filledSlots: a beatmap matching
	// multiple rules counts toward every matched skillset, but only once
	// toward filledSlots. Skillset shares therefore do not partition filled
	// slots the way CompositionAnalyzer's category counts do.
	skillsetCounts := map[Skillset]int{}

	for _, c := range stage.Categories {
		for _, slot := range c.Slots {
			if slot.Beatmap == nil {
				continue
			}
			filledSlots++
			profile := pattern.ComputeSkillsetProfile(slot.Beatmap)

			matched := false
			for _, rule := range taxonomy {
				if rule.Match(profile) {
					skillsetCounts[rule.Skillset]++
					matched = true
				}
			}
			if !matched {
				skillsetCounts[skillsetUnclassified]++
			}
		}
	}

	metrics := map[string]float64{
		"filled_slots":       float64(filledSlots),
		"distinct_skillsets": float64(len(skillsetCounts)),
	}
	if filledSlots == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	var findings []domain.Finding

	sortedSkillsets := make([]Skillset, 0, len(skillsetCounts))
	for s := range skillsetCounts {
		sortedSkillsets = append(sortedSkillsets, s)
	}
	sort.Slice(sortedSkillsets, func(i, j int) bool { return sortedSkillsets[i] < sortedSkillsets[j] })

	maxShare, maxSkillset := 0.0, Skillset("")
	for _, s := range sortedSkillsets {
		count := skillsetCounts[s]
		metrics["skillset_count_"+string(s)] = float64(count)
		share := float64(count) / float64(filledSlots)
		if share > maxShare {
			maxShare = share
			maxSkillset = s
		}
	}
	metrics["max_skillset_share"] = maxShare

	if maxSkillset != "" && maxSkillset != skillsetUnclassified && maxShare > skillsetMajorityThreshold {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("skillset %q accounts for %.0f%% of this stage's filled slots", maxSkillset, maxShare*100),
			Reason:         "a pool skewed toward one skillset systematically disadvantages players weak in that one area every match, regardless of how varied the pool's difficulty numbers are",
			Recommendation: "select at least one beatmap from an under-represented skillset for this stage",
		})
	}

	if filledSlots >= minSlotsForCoverageJudgment {
		var missing []string
		for _, rule := range taxonomy {
			if skillsetCounts[rule.Skillset] == 0 {
				missing = append(missing, string(rule.Skillset))
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("this stage has no beatmap covering skillset(s): %s", strings.Join(missing, ", ")),
				Reason:         "a taxonomy skillset with zero representation is a mechanical skill this stage never tests, regardless of how many maps or categories it has",
				Recommendation: "select at least one beatmap covering each missing skillset for this stage",
			})
		}
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = SkillCoverageAnalyzer{}
