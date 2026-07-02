package tournament

import (
	"context"
	"fmt"
	"sort"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
)

// Thresholds behind DifficultySpreadAnalyzer. Named individually, same
// convention as spikeMultiplier/jumpDistanceThreshold elsewhere in this
// package — stated conventions for explainability, not measured facts.
const (
	// gapMultiplier flags a jump between two consecutive slots' Star
	// Rating as a "gap" when it exceeds this multiple of the stage's
	// median consecutive gap.
	gapMultiplier = 2.0

	// spikeDeviationThreshold (stars) flags a slot as a local outlier when
	// its Star Rating deviates from the mean of its immediate neighbors by
	// more than this amount.
	spikeDeviationThreshold = 1.0

	// minExpectedSpread (stars) is the smallest max-min Star Rating spread
	// a stage with enough slots is expected to have; below this, the
	// stage's difficulty progression reads as "too tight" (every map
	// roughly the same difficulty).
	minExpectedSpread = 0.3

	// wideSpreadMultiplier flags a stage's overall spread as "too wide"
	// when it exceeds this multiple of the stage's own median consecutive
	// delta — self-referential, since no cross-tournament "expected
	// spread" norm exists yet (deferred follow-up).
	wideSpreadMultiplier = 3.0

	// projectedDeviationThreshold (stars) flags the stage's mean usable
	// Star Rating as deviating too far from its projected target.
	projectedDeviationThreshold = 0.5

	// minSlotsForSpreadJudgment is the fewest usable slots a stage needs
	// before "too tight" is a meaningful finding rather than noise from a
	// stage that simply doesn't have enough maps yet (mirrors
	// SkillCoverageAnalyzer's minSlotsForCoverageJudgment).
	minSlotsForSpreadJudgment = 3
)

// StarRatingLookup is the data dependency DifficultySpreadAnalyzer needs:
// real, mod-specific Star Rating for one beatmap. It is injected (not a
// concrete osuapi/storage import) so the analyzer stays pure and
// network-free, per every other analyzer's contract — the data behind it
// was fetched over the network at import-time enrichment
// (internal/enrich), never during analysis. storage.StarRatingRepository
// satisfies this interface as-is.
type StarRatingLookup interface {
	Find(ctx context.Context, beatmapID string, mods uint32) (*domain.StarRating, error)
	FindAllForBeatmap(ctx context.Context, beatmapID string) ([]domain.StarRating, error)
}

// DifficultySpreadAnalyzer reports, per Stage, whether filled slots have a
// fair/expected Star Rating progression — flagging gaps, local spikes,
// too-tight or too-wide overall spread, and deviation from the stage's
// projected target (EffectiveProjectedStarRating). This is the
// within-stage counterpart to ProgressionAnalyzer's across-stage,
// OD-proxy-based check: a different axis entirely (spread = within one
// stage's slot ordering; progression = stage-to-stage), so it is a
// separate analyzer rather than an extension.
//
// FreeMod ("FM") slots are not skipped: their difficulty is represented as
// a [min, max] range across modmap.FreeModCandidates (NoMod, HardRock,
// Easy — DoubleTime is excluded, not a legal FreeMod pick under this
// project's tournament convention), and a slot is only flagged as an
// anomaly when its entire range falls outside the acceptable comparison —
// an FM slot is never flagged just because one edge of its range looks
// anomalous.
type DifficultySpreadAnalyzer struct {
	StarRatings StarRatingLookup
}

func (DifficultySpreadAnalyzer) Name() string { return "difficulty-spread-analyzer" }

func (DifficultySpreadAnalyzer) ScopeType() domain.ScopeType { return domain.ScopeStage }

// spreadEntry is one usable (Star Rating known or ranged) slot in a
// stage's within-stage sequence.
type spreadEntry struct {
	categoryName string
	slotID       string
	min, max     float64 // equal for a fixed-mod slot; a genuine range for FreeMod
	isFreeMod    bool
}

func (e spreadEntry) mid() float64 { return (e.min + e.max) / 2 }

func (a DifficultySpreadAnalyzer) Analyze(ctx context.Context, in analysis.Input) (analysis.Result, error) {
	stage := analysis.FindStage(in.Tournament, in.Scope.ID)
	if stage == nil {
		return analysis.Result{}, fmt.Errorf("tournament: stage %q not found in tournament", in.Scope.ID)
	}

	categories := append([]domain.Category(nil), stage.Categories...)
	sort.SliceStable(categories, func(i, j int) bool { return categories[i].Order < categories[j].Order })

	filledSlots := 0
	skippedNoFixedMod := 0
	skippedNoSRData := 0
	fmSlotsRanged := 0
	var entries []spreadEntry

	for _, c := range categories {
		slots := append([]domain.Slot(nil), c.Slots...)
		sort.SliceStable(slots, func(i, j int) bool { return slots[i].Position < slots[j].Position })

		for _, slot := range slots {
			if slot.Beatmap == nil {
				continue
			}
			filledSlots++

			if modmap.IsFreeMod(c.Name) {
				entry, ok := a.freeModEntry(ctx, c.Name, slot)
				if !ok {
					skippedNoSRData++
					continue
				}
				fmSlotsRanged++
				entries = append(entries, entry)
				continue
			}

			mods, ok := modmap.FromCategoryName(c.Name)
			if !ok {
				skippedNoFixedMod++
				continue
			}
			sr, err := a.StarRatings.Find(ctx, slot.Beatmap.ID, uint32(mods))
			if err != nil {
				skippedNoSRData++
				continue
			}
			entries = append(entries, spreadEntry{categoryName: c.Name, slotID: slot.ID, min: sr.Value, max: sr.Value})
		}
	}

	metrics := map[string]float64{
		"filled_slots":               float64(filledSlots),
		"usable_slots":               float64(len(entries)),
		"skipped_slots_no_fixed_mod": float64(skippedNoFixedMod),
		"skipped_slots_no_sr_data":   float64(skippedNoSRData),
		"fm_slots_ranged":            float64(fmSlotsRanged),
	}

	if len(entries) == 0 {
		return analysis.Result{Metrics: metrics}, nil
	}

	mids := make([]float64, len(entries))
	for i, e := range entries {
		mids[i] = e.mid()
	}
	minMid, maxMid := rangeOf(mids)
	metrics["sr_range"] = maxMid - minMid

	var findings []domain.Finding

	// Gap detection: the distance between the closer edges of two
	// consecutive slots' ranges (0 when the ranges overlap), flagged when
	// it exceeds gapMultiplier times the stage's median consecutive gap.
	var gapDeltas []float64
	if len(entries) >= 2 {
		gapDeltas = make([]float64, len(entries)-1)
		for i := 1; i < len(entries); i++ {
			gapDeltas[i-1] = edgeDistance(entries[i-1], entries[i])
		}
	}
	medGap := medianPositive(gapDeltas)
	metrics["sr_median_delta"] = medGap

	gapCount := 0
	if medGap > 0 {
		for i, d := range gapDeltas {
			if d > gapMultiplier*medGap {
				gapCount++
				findings = append(findings, domain.Finding{
					Severity:       domain.SeverityWarning,
					Description:    fmt.Sprintf("Star Rating gap of %.2f between %q and %q, more than %.0fx this stage's typical consecutive gap (%.2f)", d, entries[i].categoryName, entries[i+1].categoryName, gapMultiplier, medGap),
					Reason:         "a disproportionately large difficulty gap between consecutive slots leaves a fairness cliff players must cross with no intermediate step",
					Recommendation: "select an intermediate-difficulty beatmap to smooth the transition, or confirm the jump is intentional for this stage",
					TargetStageID:  stage.ID,
				})
			}
		}
	}
	metrics["gap_count"] = float64(gapCount)

	// Spike detection: a slot whose range is entirely outside the mean of
	// its immediate neighbors by more than spikeDeviationThreshold.
	spikeCount := 0
	for i := 1; i < len(entries)-1; i++ {
		neighborMean := (entries[i-1].mid() + entries[i+1].mid()) / 2
		dev := pointEdgeDistance(entries[i], neighborMean)
		if dev > spikeDeviationThreshold {
			spikeCount++
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("%q's Star Rating deviates from its neighbors' average by %.2f, more than the %.2f-star spike threshold", entries[i].categoryName, dev, spikeDeviationThreshold),
				Reason:         "a slot disproportionately harder or easier than both its neighbors reads as an isolated outlier rather than part of the stage's intended progression",
				Recommendation: fmt.Sprintf("review whether %q's difficulty relative to its neighbors is intentional for this stage", entries[i].categoryName),
				TargetStageID:  stage.ID,
			})
		}
	}
	metrics["spike_count"] = float64(spikeCount)

	// Too tight / too wide overall spread.
	spread := maxMid - minMid
	if len(entries) >= minSlotsForSpreadJudgment && spread < minExpectedSpread {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityInfo,
			Description:    fmt.Sprintf("this stage's %d usable slots span only %.2f stars", len(entries), spread),
			Reason:         "a stage with several maps but almost no Star Rating variation tests the same difficulty repeatedly rather than a meaningful progression",
			Recommendation: "consider spreading slot difficulty further across this stage's Star Rating range",
			TargetStageID:  stage.ID,
		})
	}
	if medGap > 0 && spread > wideSpreadMultiplier*medGap {
		findings = append(findings, domain.Finding{
			Severity:       domain.SeverityWarning,
			Description:    fmt.Sprintf("this stage's overall Star Rating spread (%.2f) is more than %.0fx its own typical consecutive gap (%.2f)", spread, wideSpreadMultiplier, medGap),
			Reason:         "an overall spread dominated by one or two large jumps, rather than a gradual progression, suggests uneven pacing even if no single gap triggered a gap finding",
			Recommendation: "review whether this stage's difficulty progression is evenly paced across its slots",
			TargetStageID:  stage.ID,
		})
	}

	// Deviation from the stage's projected target. Deliberately does NOT
	// call EffectiveProjectedStarRating here: that helper's NM1 fallback
	// reads domain.Beatmap.StarRating, which this codebase never
	// populates (real Star Rating lives in StarRatingRepository, kept
	// separate from the immutable Beatmap aggregate — see
	// domain.StarRating's doc comment) — it would always read 0 and
	// manufacture a false deviation finding for every stage. This
	// analyzer computes its own NM1 fallback from real per-mod data
	// instead.
	if target := a.projectedTarget(ctx, *stage); target != nil {
		metrics["projected_star_rating"] = *target
		meanMid := mean(mids)
		deviation := meanMid - *target
		if deviation < 0 {
			deviation = -deviation
		}
		metrics["deviation_from_projected"] = deviation
		if deviation > projectedDeviationThreshold {
			findings = append(findings, domain.Finding{
				Severity:       domain.SeverityWarning,
				Description:    fmt.Sprintf("this stage's average usable Star Rating (%.2f) deviates from its projected target (%.2f) by %.2f", meanMid, *target, deviation),
				Reason:         "a stage whose actual difficulty diverges from its own projected target is either miscalibrated or the target needs updating",
				Recommendation: "review beatmap selection against the stage's projected Star Rating, or update the projected target if the divergence is intentional",
				TargetStageID:  stage.ID,
			})
		}
	}

	var score *float64
	if pairs := len(entries) - 1; pairs > 0 {
		anomalies := gapCount + spikeCount
		s := 1.0 - float64(anomalies)/float64(pairs)
		score = &s
	}

	return analysis.Result{Score: score, Metrics: metrics, Findings: findings}, nil
}

// projectedTarget returns the stage's explicit ProjectedStarRating if set,
// otherwise its NM1 slot's real NoMod Star Rating (from StarRatingLookup,
// not the always-zero domain.Beatmap.StarRating field EffectiveProjectedStarRating
// falls back to for API-display purposes). Returns nil if neither is
// available.
func (a DifficultySpreadAnalyzer) projectedTarget(ctx context.Context, s domain.Stage) *float64 {
	if s.ProjectedStarRating != nil {
		return s.ProjectedStarRating
	}
	for _, c := range s.Categories {
		if c.Name != "NM" {
			continue
		}
		for _, slot := range c.Slots {
			if slot.Position != 1 || slot.Beatmap == nil {
				continue
			}
			sr, err := a.StarRatings.Find(ctx, slot.Beatmap.ID, uint32(modmap.NoMod))
			if err != nil {
				return nil
			}
			v := sr.Value
			return &v
		}
	}
	return nil
}

// freeModEntry resolves a FreeMod slot's difficulty range across
// modmap.FreeModCandidates, using whichever candidates have been fetched
// for the slot's beatmap. Returns ok=false if none of the candidates have
// been fetched yet — a graceful "no SR data" skip, not an error.
func (a DifficultySpreadAnalyzer) freeModEntry(ctx context.Context, categoryName string, slot domain.Slot) (spreadEntry, bool) {
	all, err := a.StarRatings.FindAllForBeatmap(ctx, slot.Beatmap.ID)
	if err != nil || len(all) == 0 {
		return spreadEntry{}, false
	}

	byMods := make(map[modmap.Mods]float64, len(all))
	for _, sr := range all {
		byMods[modmap.Mods(sr.Mods)] = sr.Value
	}

	var values []float64
	for _, candidate := range modmap.FreeModCandidates {
		if v, ok := byMods[candidate]; ok {
			values = append(values, v)
		}
	}
	if len(values) == 0 {
		return spreadEntry{}, false
	}

	min, max := rangeOf(values)
	return spreadEntry{categoryName: categoryName, slotID: slot.ID, min: min, max: max, isFreeMod: true}, true
}

// edgeDistance returns the distance between the closer edges of a and b's
// ranges, or 0 when the ranges overlap.
func edgeDistance(a, b spreadEntry) float64 {
	if a.max < b.min {
		return b.min - a.max
	}
	if b.max < a.min {
		return a.min - b.max
	}
	return 0
}

// pointEdgeDistance returns the distance from point to e's range, or 0
// when point falls within [e.min, e.max].
func pointEdgeDistance(e spreadEntry, point float64) float64 {
	if point < e.min {
		return e.min - point
	}
	if point > e.max {
		return point - e.max
	}
	return 0
}

// medianPositive returns the median of the positive values in values,
// ignoring zeros (overlapping-range gaps) — mirrors ProgressionAnalyzer's
// positiveDeltas-then-median pattern, so a majority-overlapping stage
// doesn't drag its gap baseline down to 0 and make every real gap look
// artificially large by comparison.
func medianPositive(values []float64) float64 {
	var positive []float64
	for _, v := range values {
		if v > 0 {
			positive = append(positive, v)
		}
	}
	return median(positive)
}

// mean returns the arithmetic mean of values, or 0 for an empty slice.
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

var _ analysis.Analyzer = DifficultySpreadAnalyzer{}
