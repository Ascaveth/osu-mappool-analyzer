package api

import (
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
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
	Name       string              `json:"name"`
	Order      int                 `json:"order"`
	Categories []categoryConfigDTO `json:"categories"`
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
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Order      int           `json:"order"`
	Categories []categoryDTO `json:"categories"`
}

type categoryDTO struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Order int       `json:"order"`
	Slots []slotDTO `json:"slots"`
}

type slotDTO struct {
	ID        string  `json:"id"`
	Position  int     `json:"position"`
	BeatmapID *string `json:"beatmap_id"`
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

func toTournamentDTO(t *domain.Tournament) tournamentDTO {
	stages := make([]stageDTO, len(t.Stages))
	for i, s := range t.Stages {
		stages[i] = toStageDTO(s)
	}
	return tournamentDTO{ID: t.ID, Name: t.Name, Edition: t.Edition, Stages: stages}
}

func toStageDTO(s domain.Stage) stageDTO {
	cats := make([]categoryDTO, len(s.Categories))
	for i, c := range s.Categories {
		cats[i] = toCategoryDTO(c)
	}
	return stageDTO{ID: s.ID, Name: s.Name, Order: s.Order, Categories: cats}
}

func toCategoryDTO(c domain.Category) categoryDTO {
	slots := make([]slotDTO, len(c.Slots))
	for i, s := range c.Slots {
		slots[i] = toSlotDTO(s)
	}
	return categoryDTO{ID: c.ID, Name: c.Name, Order: c.Order, Slots: slots}
}

func toSlotDTO(s domain.Slot) slotDTO {
	var beatmapID *string
	if s.Beatmap != nil {
		id := s.Beatmap.ID
		beatmapID = &id
	}
	return slotDTO{ID: s.ID, Position: s.Position, BeatmapID: beatmapID}
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
		stages[i] = domain.Stage{Name: s.Name, Order: s.Order, Categories: cats}
	}
	return &domain.Tournament{Name: dto.Name, Edition: dto.Edition, Stages: stages}
}
