// Command server is the composition root for the osu! Mappool Analyzer API:
// it wires the in-memory repositories, registers every Analyzer with the
// Analysis Engine, and serves docs/api/openapi.yaml over HTTP.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/metadata"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/pattern"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/analysis/tournament"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/api"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/config"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/enrich"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/osuapi"
	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/storage/memory"
)

func main() {
	// Loads backend/.env into the process environment for local
	// development convenience. A missing file is not an error — real
	// deployments (Railway, etc.) set env vars directly and have no .env
	// file at all; only a malformed .env file that does exist is fatal.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("loading .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	starRatings := memory.NewStarRatingRepository()

	engine := analysis.NewEngine()
	for _, a := range []analysis.Analyzer{
		metadata.BPMRangeAnalyzer{},
		metadata.DifficultySettingsAnalyzer{},
		metadata.ARCalibrationAnalyzer{},
		metadata.CSPrecisionAnalyzer{},
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
		tournament.SkillRedundancyAnalyzer{},
		tournament.DifficultySpreadAnalyzer{StarRatings: starRatings},
	} {
		if err := engine.Register(a); err != nil {
			log.Fatalf("registering analyzer %q: %v", a.Name(), err)
		}
	}

	// DifficultySpreadAnalyzer is registered regardless of whether star
	// rating fetching is enabled: it degrades gracefully to metrics-only
	// results when no Star Rating data exists for a stage's beatmaps,
	// same as every other analyzer's "insufficient data" guard.
	var enricher api.Enricher
	if cfg.StarRatingFetchEnabled {
		enricher = &enrich.Enricher{
			OsuAPI:      osuapi.New(cfg.OsuClientID, cfg.OsuClientSecret),
			StarRatings: starRatings,
		}
	} else {
		log.Print("star rating enrichment disabled: OSU_CLIENT_ID/OSU_CLIENT_SECRET not set")
	}

	server := api.NewServer(
		memory.NewTournamentRepository(),
		memory.NewBeatmapRepository(),
		engine,
		enricher,
	)

	handler := api.Logging(api.CORS(cfg.AllowedOrigins, api.NewRouter(server)))

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("osu-mappool-analyzer backend listening on :%s", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
