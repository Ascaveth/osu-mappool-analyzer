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

// redundancySimilarityThreshold marks two beatmaps' normalized skill
// profiles as near-identical. Distance is a 0-1 RMS across six min-max
// normalized dimensions (see skillProfileVector), so 0 means literally
// identical and 1 means maximally spread within this stage. Named and
// documented individually, following the same convention as
// jumpDistanceThreshold/techAnchorCountThreshold in skillcoverage.go, since
// this is a stated calibration choice, not a measured fact.
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
type skillProfileVector [6]float64

func vectorFromProfile(p pattern.SkillsetProfile) skillProfileVector {
	return skillProfileVector{
		p.AvgJumpDistance,
		float64(p.StreamCount),
		float64(p.LongestRunLength),
		p.AvgAnchorCount,
		p.ReverseSliderRatio,
		p.SpinnerDensity,
	}
}

// normalizeVectors min-max normalizes each dimension across the given
// vectors, relative to this stage only — the same principle
// BalanceAnalyzer already applies to AR/OD/slider-ratio variance, since
// what counts as a "wide" jump or "dense" stream is relative to the pool
// being built, not an absolute constant. A dimension with zero range
// across every vector (e.g. no beatmap in the stage uses sliders) carries
// no distinguishing information, so it is left at zero for every vector
// rather than divided by zero.
func normalizeVectors(vectors []skillProfileVector) []skillProfileVector {
	if len(vectors) == 0 {
		return nil
	}
	var mins, maxs skillProfileVector
	mins = vectors[0]
	maxs = vectors[0]
	for _, v := range vectors[1:] {
		for i := range v {
			if v[i] < mins[i] {
				mins[i] = v[i]
			}
			if v[i] > maxs[i] {
				maxs[i] = v[i]
			}
		}
	}

	normalized := make([]skillProfileVector, len(vectors))
	for i, v := range vectors {
		var n skillProfileVector
		for d := range v {
			rangeD := maxs[d] - mins[d]
			if rangeD <= 0 {
				n[d] = 0
				continue
			}
			n[d] = (v[d] - mins[d]) / rangeD
		}
		normalized[i] = n
	}
	return normalized
}

// rmsDistance is the root-mean-square distance between two normalized
// vectors: 0 means identical on every dimension, 1 means maximally
// different on every dimension.
func rmsDistance(a, b skillProfileVector) float64 {
	sumSq := 0.0
	for i := range a {
		d := a[i] - b[i]
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(a)))
}

type redundantPair struct {
	slotAID, slotBID     string
	categoryA, categoryB string
	distance             float64
}

func (SkillRedundancyAnalyzer) Analyze(_ context.Context, in analysis.Input) (analysis.Result, error) {
	stage := analysis.FindStage(in.Tournament, in.Scope.ID)
	if stage == nil {
		return analysis.Result{}, fmt.Errorf("tournament: stage %q not found in tournament", in.Scope.ID)
	}

	var slotIDs []string
	var categoryNames []string
	var vectors []skillProfileVector
	for _, c := range stage.Categories {
		for _, slot := range c.Slots {
			if slot.Beatmap == nil {
				continue
			}
			slotIDs = append(slotIDs, slot.ID)
			categoryNames = append(categoryNames, c.Name)
			vectors = append(vectors, vectorFromProfile(pattern.ComputeSkillsetProfile(slot.Beatmap)))
		}
	}

	metrics := map[string]float64{"filled_slots": float64(len(slotIDs))}
	if len(slotIDs) < minSlotsForRedundancyJudgment {
		return analysis.Result{Metrics: metrics}, nil
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
				pairs = append(pairs, redundantPair{
					slotAID: slotIDs[i], slotBID: slotIDs[j],
					categoryA: categoryNames[i], categoryB: categoryNames[j],
					distance: d,
				})
			}
		}
	}

	metrics["redundant_pair_count"] = float64(len(pairs))
	metrics["closest_pair_distance"] = closest

	if len(pairs) == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].distance != pairs[j].distance {
			return pairs[i].distance < pairs[j].distance
		}
		if pairs[i].slotAID != pairs[j].slotAID {
			return pairs[i].slotAID < pairs[j].slotAID
		}
		return pairs[i].slotBID < pairs[j].slotBID
	})

	reportCount := len(pairs)
	if reportCount > maxRedundancyFindings {
		reportCount = maxRedundancyFindings
	}

	findings := make([]domain.Finding, 0, reportCount)
	for _, p := range pairs[:reportCount] {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("slot %q (%s) and slot %q (%s) test a near-identical mechanical skill profile despite different category labels", p.slotAID, p.categoryA, p.slotBID, p.categoryB),
			Reason:         "a map can be individually well-built and correctly calibrated yet still burn a slot the pool didn't need, if it tests the same skill the same way another slot already does",
			Recommendation: "replace one of these beatmaps with a map that covers a different mechanical skill, or a different way of testing the same skill",
		})
	}

	return analysis.Result{Metrics: metrics, Findings: findings}, nil
}

var _ analysis.Analyzer = SkillRedundancyAnalyzer{}
