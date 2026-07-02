// Command server is the composition root for the osu! Mappool Analyzer API:
// it wires the in-memory repositories, registers every Analyzer with the
// Analysis Engine, and serves docs/api/openapi.yaml over HTTP.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/metadata"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/pattern"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/tournament"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/api"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/memory"
)

func main() {
	engine := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		metadata.BPMRangeAnalyzer{},
		metadata.DifficultySettingsAnalyzer{},
		metadata.MapperRepetitionAnalyzer{},
		metadata.ObjectDensityAnalyzer{},
		pattern.JumpDistanceAnalyzer{},
		pattern.JumpAngleAnalyzer{},
		pattern.SliderComplexityAnalyzer{},
		pattern.SpinnerUsageAnalyzer{},
		pattern.StreamBurstAnalyzer{},
		tournament.CompositionAnalyzer{},
		tournament.ProgressionAnalyzer{},
		tournament.BalanceAnalyzer{},
		tournament.DiversityAnalyzer{},
		tournament.SkillCoverageAnalyzer{},
	} {
		if err := engine.Register(a); err != nil {
			log.Fatalf("registering analyzer %q: %v", a.Name(), err)
		}
	}

	server := api.NewServer(
		memory.NewTournamentRepository(),
		memory.NewBeatmapRepository(),
		engine,
	)

	handler := api.Logging(api.CORS(allowedOrigins(), api.NewRouter(server)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("osu-mappool-analyzer backend listening on :%s", port)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func allowedOrigins() []string {
	raw := os.Getenv("ALLOWED_ORIGINS")
	if raw == "" {
		return []string{"http://localhost:3000"}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
