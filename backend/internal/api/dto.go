package api

import (
	"context"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/tournament"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/modmap"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage"
)

// Wire request/response shapes, matching docs/api/openapi.yaml's schemas
// exactly (snake_case JSON, no field this file invents that the spec
// doesn't already define).

type tournamentConfigurationDTO struct {
	Name    string           `json:"name"`
	Edition string           `json:"edition"`
	Stages  []stageConfigDTO `json:"stages"`
}

type stageConfigDTO struct {
	Name                string              `json:"name"`
	Order               int                 `json:"order"`
	Categories          []categoryConfigDTO `json:"categories"`
	ProjectedStarRating *float64            `json:"projectedStarRating"`
}

type categoryConfigDTO struct {
	Name      string `json:"name"`
	Order     int    `json:"order"`
	SlotCount int    `json:"slotCount"`
}

type tournamentUpdateDTO struct {
	Name    *string `json:"name"`
	Edition *string `json:"edition"`
}

type tournamentSummaryDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Edition    string `json:"edition"`
	StageCount int    `json:"stage_count"`
}

type tournamentDTO struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Edition string     `json:"edition"`
	Stages  []stageDTO `json:"stages"`
}

type stageDTO struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Order               int           `json:"order"`
	Categories          []categoryDTO `json:"categories"`
	ProjectedStarRating *float64      `json:"projected_star_rating"`
}

type categoryDTO struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Order int       `json:"order"`
	Slots []slotDTO `json:"slots"`
}

type slotDTO struct {
	ID                  string                  `json:"id"`
	Position            int                     `json:"position"`
	BeatmapID           *string                 `json:"beatmap_id"`
	EffectiveDifficulty *effectiveDifficultyDTO `json:"effective_difficulty"`
}

// effectiveDifficultyDTO is a filled slot's AR/OD/CS/HP/BPM/length as they
// actually play under the slot's own Category's fixed mod (HR, DT, ...),
// not the beatmap's raw .osu values — see modmap.EffectiveDifficultyFor.
// Distinct from Beatmap's own raw fields (never duplicated: Slot still
// references Beatmap by ID only, per docs/06-domain-model.md) because this
// is placement-specific derived data, not a second copy of the Beatmap
// aggregate — the same beatmap re-used in an HR slot and an NM slot has
// two different EffectiveDifficulty values but one Beatmap.
type effectiveDifficultyDTO struct {
	AR            float64  `json:"ar"`
	OD            float64  `json:"od"`
	CS            float64  `json:"cs"`
	HP            float64  `json:"hp"`
	BPM           float64  `json:"bpm"`
	LengthSeconds int      `json:"length_seconds"`
	StarRating    *float64 `json:"star_rating"`
}

type beatmapDTO struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Artist        string   `json:"artist"`
	Mapper        string   `json:"mapper"`
	Version       string   `json:"version"`
	Tags          []string `json:"tags"`
	AR            float64  `json:"ar"`
	OD            float64  `json:"od"`
	CS            float64  `json:"cs"`
	HP            float64  `json:"hp"`
	BPM           float64  `json:"bpm"`
	StarRating    float64  `json:"star_rating"`
	LengthSeconds int      `json:"length_seconds"`
	ObjectCount   int      `json:"object_count"`
	SliderRatio   float64  `json:"slider_ratio"`
	OsuFileHash   string   `json:"osu_file_hash"`
}

type scopeDTO struct {
	Type domain.ScopeType `json:"type"`
	ID   string           `json:"id"`
}

type findingDTO struct {
	Severity       domain.Severity    `json:"severity"`
	Description    string             `json:"description"`
	Reason         string             `json:"reason"`
	Recommendation string             `json:"recommendation"`
	Metrics        map[string]float64 `json:"metrics,omitempty"`
}

type analysisDTO struct {
	AnalyzerName string             `json:"analyzer_name"`
	Scope        scopeDTO           `json:"scope"`
	SourceHash   string             `json:"source_hash"`
	GeneratedAt  time.Time          `json:"generated_at"`
	Score        *float64           `json:"score"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
	Findings     []findingDTO       `json:"findings"`
}

type citationDTO struct {
	AnalyzerName string     `json:"analyzer_name"`
	Scope        scopeDTO   `json:"scope"`
	Finding      findingDTO `json:"finding"`
}

type reportSectionsDTO struct {
	Summary         string             `json:"summary"`
	Findings        []citationDTO      `json:"findings"`
	Warnings        []citationDTO      `json:"warnings"`
	Recommendations []string           `json:"recommendations"`
	Statistics      map[string]float64 `json:"statistics"`
}

type reportDTO struct {
	Scope       scopeDTO          `json:"scope"`
	GeneratedAt time.Time         `json:"generated_at"`
	Sections    reportSectionsDTO `json:"sections"`
}

// --- domain -> wire ---

func toTournamentSummaryDTO(t domain.Tournament) tournamentSummaryDTO {
	return tournamentSummaryDTO{
		ID:         t.ID,
		Name:       t.Name,
		Edition:    t.Edition,
		StageCount: len(t.Stages),
	}
}

func toTournamentDTO(ctx context.Context, t *domain.Tournament, starRatings storage.StarRatingRepository) tournamentDTO {
	stages := make([]stageDTO, len(t.Stages))
	for i, s := range t.Stages {
		stages[i] = toStageDTO(ctx, s, starRatings)
	}
	return tournamentDTO{ID: t.ID, Name: t.Name, Edition: t.Edition, Stages: stages}
}

func toStageDTO(ctx context.Context, s domain.Stage, starRatings storage.StarRatingRepository) stageDTO {
	cats := make([]categoryDTO, len(s.Categories))
	for i, c := range s.Categories {
		cats[i] = toCategoryDTO(ctx, c, starRatings)
	}
	return stageDTO{
		ID:                  s.ID,
		Name:                s.Name,
		Order:               s.Order,
		Categories:          cats,
		ProjectedStarRating: tournament.EffectiveProjectedStarRating(s),
	}
}

func toCategoryDTO(ctx context.Context, c domain.Category, starRatings storage.StarRatingRepository) categoryDTO {
	slots := make([]slotDTO, len(c.Slots))
	for i, s := range c.Slots {
		slots[i] = toSlotDTO(ctx, s, c.Name, starRatings)
	}
	return categoryDTO{ID: c.ID, Name: c.Name, Order: c.Order, Slots: slots}
}

func toSlotDTO(ctx context.Context, s domain.Slot, categoryName string, starRatings storage.StarRatingRepository) slotDTO {
	var beatmapID *string
	if s.Beatmap != nil {
		id := s.Beatmap.ID
		beatmapID = &id
	}
	return slotDTO{
		ID:                  s.ID,
		Position:            s.Position,
		BeatmapID:           beatmapID,
		EffectiveDifficulty: toEffectiveDifficultyDTO(ctx, s.Beatmap, categoryName, starRatings),
	}
}

// toEffectiveDifficultyDTO returns nil when the slot is unfilled or its
// Category has no single fixed mod (FreeMod, Tiebreaker, or an
// unrecognized name) — there is no sound single mod-adjusted value to
// report in either case (see modmap.FromCategoryName). Its StarRating
// field is separately best-effort: nil starRatings (fetching disabled) or
// a lookup miss (not yet enriched) both leave StarRating nil without
// failing the rest of the effective-difficulty payload, same as every
// analyzer's "insufficient data" convention.
func toEffectiveDifficultyDTO(ctx context.Context, b *domain.Beatmap, categoryName string, starRatings storage.StarRatingRepository) *effectiveDifficultyDTO {
	if b == nil {
		return nil
	}
	mods, ok := modmap.FromCategoryName(categoryName)
	if !ok {
		return nil
	}
	eff := modmap.EffectiveDifficultyFor(b.AR, b.OD, b.CS, b.HP, b.BPM, b.LengthSeconds, mods)
	return &effectiveDifficultyDTO{
		AR: eff.AR, OD: eff.OD, CS: eff.CS, HP: eff.HP, BPM: eff.BPM, LengthSeconds: eff.LengthSeconds,
		StarRating: lookupStarRating(ctx, b.ID, mods, starRatings),
	}
}

func lookupStarRating(ctx context.Context, beatmapID string, mods modmap.Mods, starRatings storage.StarRatingRepository) *float64 {
	if starRatings == nil {
		return nil
	}
	sr, err := starRatings.Find(ctx, beatmapID, uint32(mods))
	if err != nil {
		return nil
	}
	v := sr.Value
	return &v
}

func toBeatmapDTO(b *domain.Beatmap) beatmapDTO {
	return beatmapDTO{
		ID:            b.ID,
		Title:         b.Title,
		Artist:        b.Artist,
		Mapper:        b.Mapper,
		Version:       b.Version,
		Tags:          b.Tags,
		AR:            b.AR,
		OD:            b.OD,
		CS:            b.CS,
		HP:            b.HP,
		BPM:           b.BPM,
		StarRating:    b.StarRating,
		LengthSeconds: b.LengthSeconds,
		ObjectCount:   b.ObjectCount,
		SliderRatio:   b.SliderRatio,
		OsuFileHash:   b.OsuFileHash,
	}
}

func toFindingDTO(f domain.Finding) findingDTO {
	return findingDTO{
		Severity:       f.Severity,
		Description:    f.Description,
		Reason:         f.Reason,
		Recommendation: f.Recommendation,
		Metrics:        f.Metrics,
	}
}

func toScopeDTO(s domain.Scope) scopeDTO {
	return scopeDTO{Type: s.Type, ID: s.ID}
}

func toAnalysisDTO(a domain.Analysis) analysisDTO {
	findings := make([]findingDTO, len(a.Findings))
	for i, f := range a.Findings {
		findings[i] = toFindingDTO(f)
	}
	return analysisDTO{
		AnalyzerName: a.AnalyzerName,
		Scope:        toScopeDTO(a.Scope),
		SourceHash:   a.SourceHash,
		GeneratedAt:  a.GeneratedAt,
		Score:        a.Score,
		Metrics:      a.Metrics,
		Findings:     findings,
	}
}

func toCitationDTO(c domain.Citation) citationDTO {
	return citationDTO{
		AnalyzerName: c.AnalyzerName,
		Scope:        toScopeDTO(c.Scope),
		Finding:      toFindingDTO(c.Finding),
	}
}

func toReportDTO(r domain.Report) reportDTO {
	findings := make([]citationDTO, len(r.Sections.Findings))
	for i, c := range r.Sections.Findings {
		findings[i] = toCitationDTO(c)
	}
	warnings := make([]citationDTO, len(r.Sections.Warnings))
	for i, c := range r.Sections.Warnings {
		warnings[i] = toCitationDTO(c)
	}
	recommendations := r.Sections.Recommendations
	if recommendations == nil {
		recommendations = []string{}
	}
	return reportDTO{
		Scope:       scopeDTO{Type: r.ScopeType, ID: r.ScopeID},
		GeneratedAt: r.GeneratedAt,
		Sections: reportSectionsDTO{
			Summary:         r.Sections.Summary,
			Findings:        findings,
			Warnings:        warnings,
			Recommendations: recommendations,
			Statistics:      r.Sections.Statistics,
		},
	}
}

// --- wire -> domain ---

func (dto tournamentConfigurationDTO) toDomain() *domain.Tournament {
	stages := make([]domain.Stage, len(dto.Stages))
	for i, s := range dto.Stages {
		cats := make([]domain.Category, len(s.Categories))
		for j, c := range s.Categories {
			slots := make([]domain.Slot, c.SlotCount)
			for k := range slots {
				slots[k] = domain.Slot{Position: k + 1}
			}
			cats[j] = domain.Category{Name: c.Name, Order: c.Order, Slots: slots}
		}
		stages[i] = domain.Stage{Name: s.Name, Order: s.Order, Categories: cats, ProjectedStarRating: s.ProjectedStarRating}
	}
	return &domain.Tournament{Name: dto.Name, Edition: dto.Edition, Stages: stages}
}
