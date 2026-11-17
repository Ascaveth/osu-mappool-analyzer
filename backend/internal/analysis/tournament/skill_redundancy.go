package tournament

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/pattern"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// redundancySimilarityThreshold is a provisional heuristic, pending pooler calibration.
const redundancySimilarityThreshold = 0.15

// maxRedundancyFindings caps how many redundant pairs are reported per
// stage. A large pool (15+ filled slots) can produce many close pairs;
// reporting only the closest ones keeps Findings actionable instead of
// spamming every near-duplicate combination.
const maxRedundancyFindings = 5

// minSlotsForRedundancyJudgment mirrors
// SkillCoverageAnalyzer.minSlotsForCoverageJudgment's rationale: fewer than
// two filled slots means there is nothing to compare, and this analyzer
// needs at least a small stage before pairwise comparison is meaningful
// rather than noise.
const minSlotsForRedundancyJudgment = 2

// SkillRedundancyAnalyzer reports, per Stage, pairs of filled slots whose
// underlying mechanical skill profile is near-identical even when their
// Category (mod/intent) labels differ — e.g. an NM1, HR1, and FM1 that all
// reduce to the same conventional-aim jump pattern despite carrying three
// different labels.
//
// This is a different failure mode than SkillCoverageAnalyzer's discrete
// taxonomy tags: two beatmaps can both match SkillCoverageAnalyzer's "aim"
// rule while having very different jump distances, stream density, and
// slider complexity — the boolean bucket can't see that two maps are
// near-duplicates on the underlying continuous profile. Named "pool-level
// redundant" in mappool-feedback-evaluation.md §1 (case 4) and §2: a map
// can be individually well-built and correctly difficulty-calibrated, yet
// still burn a slot the pool didn't need to spend, because it tests the
// same skill the same way another slot already does.
type SkillRedundancyAnalyzer struct{}

func (SkillRedundancyAnalyzer) Name() string { return "skill-redundancy-analyzer" }

func (SkillRedundancyAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeStage }

// skillProfileVector is the subset of pattern.SkillsetProfile's continuous
// fields used for redundancy comparison. MaxJumpDistance and ObjectCount
// are deliberately excluded: MaxJumpDistance is redundant with
// AvgJumpDistance for this purpose, and ObjectCount describes beatmap
// length, not mechanical skill.
type skillProfileVector [7]float64

func vectorFromProfile(p pattern.SkillsetProfile) skillProfileVector {
	return skillProfileVector{
		p.AvgJumpDistance,
		float64(p.StreamCount),
		float64(p.LongestRunLength),
		p.AvgAnchorCount,
		p.ReverseSliderRatio,
		p.SpinnerDensity,
		p.BPM,
	}
}

// Fixed feature scales keep pair distances independent of stage composition.
// shortcut: scales are provisional heuristics; revisit with Task 3 features and Task 5 calibration.
// Units: osu pixels, runs, notes, anchors, ratio, ratio, BPM.
var redundancyFeatureScales = skillProfileVector{200, 10, 32, 4, 1, 1, 100}

func normalizeVectors(vectors []skillProfileVector) []skillProfileVector {
	normalized := make([]skillProfileVector, len(vectors))
	for i, v := range vectors {
		for d := range v {
			normalized[i][d] = v[d] / redundancyFeatureScales[d]
		}
	}
	return normalized
}

// rmsDistance is the root-mean-square distance between two normalized
// vectors: 0 means identical; distances are not bounded by 1.
func rmsDistance(a, b skillProfileVector) float64 {
	sumSq := 0.0
	for i := range a {
		d := a[i] - b[i]
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(a)))
}

type redundantPair struct {
	slotACode, slotBCode string
	distance             float64
	evidence             string
}

func (SkillRedundancyAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	stage := analysis.FindStage(in.Tournament, in.Scope.ID)
	if stage == nil {
		return analysis.Result{}, fmt.Errorf("tournament: stage %q not found in tournament", in.Scope.ID)
	}

	nameOccurrences := make(map[string]int, len(stage.Categories))
	for _, c := range stage.Categories {
		nameOccurrences[c.Name]++
	}

	skippedSlots := 0
	var slotCodes []string
	var vectors []skillProfileVector
	for _, c := range stage.Categories {
		for _, slot := range c.Slots {
			if slot.Beatmap == nil {
				continue
			}
			profile, ok := pattern.ComputePlacementSkillsetProfile(slot.Beatmap, c.Name)
			if !ok {
				skippedSlots++
				continue
			}
			code := fmt.Sprintf("%s%d", c.Name, slot.Position)
			if nameOccurrences[c.Name] > 1 {
				code = fmt.Sprintf("%s%d (category #%d)", c.Name, slot.Position, c.Order)
			}
			slotCodes = append(slotCodes, code)
			vectors = append(vectors, vectorFromProfile(profile))
		}
	}

	metrics := map[string]float64{"filled_slots": float64(len(slotCodes) + skippedSlots), "analyzed_slots": float64(len(slotCodes)), "skipped_slots_no_fixed_mod": float64(skippedSlots)}
	if len(slotCodes) < minSlotsForRedundancyJudgment {
		return analysis.Result{Metrics: metrics, Findings: partialSkillFindings(skippedSlots)}, nil
	}

	normalized := normalizeVectors(vectors)

	var pairs []redundantPair
	closest := math.Inf(1)
	for i := 0; i < len(normalized); i++ {
		for j := i + 1; j < len(normalized); j++ {
			d := rmsDistance(normalized[i], normalized[j])
			if d < closest {
				closest = d
			}
			if d < redundancySimilarityThreshold {
				a, b := slotCodes[i], slotCodes[j]
				if a > b {
					a, b = b, a
				}
				pairs = append(pairs, redundantPair{
					slotACode: a, slotBCode: b,
					evidence: fmt.Sprintf("Scaled feature differences (absolute delta / provisional scale): jump distance %.3f, stream count %.3f, longest run %.3f, anchors %.3f, reverse ratio %.3f, spinner density %.3f, effective BPM %.3f; RMS %.4f < %.2f. Scales and threshold are uncalibrated heuristics.", math.Abs(normalized[i][0]-normalized[j][0]), math.Abs(normalized[i][1]-normalized[j][1]), math.Abs(normalized[i][2]-normalized[j][2]), math.Abs(normalized[i][3]-normalized[j][3]), math.Abs(normalized[i][4]-normalized[j][4]), math.Abs(normalized[i][5]-normalized[j][5]), math.Abs(normalized[i][6]-normalized[j][6]), d, redundancySimilarityThreshold),
					distance: d,
				})
			}
		}
	}

	metrics["redundant_pair_count"] = float64(len(pairs))
	metrics["closest_pair_distance"] = closest

	if len(pairs) == 0 {
		return analysis.Result{Metrics: metrics, Findings: partialSkillFindings(skippedSlots)}, nil
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].distance != pairs[j].distance {
			return pairs[i].distance < pairs[j].distance
		}
		if pairs[i].slotACode != pairs[j].slotACode {
			return pairs[i].slotACode < pairs[j].slotACode
		}
		return pairs[i].slotBCode < pairs[j].slotBCode
	})

	reportCount := len(pairs)
	if reportCount > maxRedundancyFindings {
		reportCount = maxRedundancyFindings
	}

	findings := partialSkillFindings(skippedSlots)
	for _, p := range pairs[:reportCount] {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("%s and %s have similar measured mechanical features", p.slotACode, p.slotBCode),
			Reason:         p.evidence,
			Recommendation: "review whether these maps serve distinct intended roles; similarity alone does not establish that either map should be replaced",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = SkillRedundancyAnalyzer{}
