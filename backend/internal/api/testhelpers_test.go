package api

import (
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/tournament"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/memory"
)

func newTestServer() *Server {
	engine := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		tournament.CompositionAnalyzer{},
		tournament.ProgressionAnalyzer{},
		tournament.BalanceAnalyzer{},
		tournament.DiversityAnalyzer{},
	} {
		_ = engine.Register(a)
	}
	return NewServer(memory.NewTournamentRepository(), memory.NewBeatmapRepository(), engine)
}

const exampleTournamentConfigJSON = `{
  "name": "Example Open",
  "edition": "2026",
  "stages": [
    {
      "name": "Qualifiers",
      "order": 1,
      "categories": [
        { "name": "NM", "order": 1, "slotCount": 5 },
        { "name": "HD", "order": 2, "slotCount": 2 }
      ]
    }
  ]
}`
