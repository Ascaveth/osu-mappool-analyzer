// Package analysis is the Analysis Engine: the plugin host that runs
// independent Analyzer implementations against a normalized Tournament
// aggregate and produces domain.Analysis results. See
// docs/09-analysis-engine-specification.md.
package analysis

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Ascaveth/osu-mappool-analyzer/backend/internal/domain"
)

// Input is what an Analyzer receives: the full normalized Tournament
// aggregate plus the specific Scope this run is responsible for.
// Analyzers receive the whole tournament (not just their scope's subtree)
// because some findings are inherently relational — e.g. a Stage-scoped
// progression analyzer needs to see neighboring stages to detect a
// difficulty spike. Analyzers are still expected to honor their declared
// ScopeType and not produce findings about unrelated scopes.
type Input struct {
	Tournament *domain.Tournament
	Scope      domain.Scope
}

// Result is what an Analyzer returns for one Input. The Engine wraps a
// Result into a domain.Analysis, attaching identity, scope, timestamp,
// and a content hash.
type Result struct {
	// Score is an optional 0.0-1.0 quality signal; nil if not applicable
	// to this analyzer.
	Score    *float64
	Metrics  map[string]float64
	Findings []domain.Finding
}

// Analyzer is a single-responsibility plugin. Implementations must never
// call or depend on another Analyzer — each one is independently testable
// against synthetic Input data (docs/04 Architecture Principle 11).
//
// ScopeType declares which kind of node this analyzer runs against; the
// Engine calls Analyze once per matching node found in the Tournament
// (e.g. a ScopeStage analyzer runs once per Stage).
type Analyzer interface {
	// Name uniquely identifies this analyzer, e.g. "composition-analyzer".
	// It is part of the Analysis's identity and its SourceHash input, so
	// renaming an analyzer is a breaking change to historical Analyses.
	Name() string

	ScopeType() domain.ScopeType

	Analyze(ctx context.Context, input Input) (Result, error)
}

// Engine holds a registry of Analyzers and runs them against a Tournament.
type Engine struct {
	analyzers map[string]Analyzer
	order     []string // registration order, for deterministic iteration

	// Now is the clock used to timestamp generated Analyses. Defaults to
	// time.Now; overridable in tests for deterministic assertions.
	Now func() time.Time

	// cacheMu guards cache. SourceHash uniquely identifies an (analyzer,
	// scope, content) triple (docs/04 Architecture Principle 6), so a
	// cache hit is guaranteed to be a reproduction of what Analyze would
	// return — this turns repeated requests against unchanged tournament
	// data from a full analyzer re-run into a hash lookup.
	cacheMu sync.RWMutex
	cache   map[string]domain.Analysis
}

// NewEngine returns an empty Engine ready for analyzer registration.
func NewEngine() *Engine {
	return &Engine{
		analyzers: map[string]Analyzer{},
		Now:       time.Now,
		cache:     map[string]domain.Analysis{},
	}
}

// Register adds an Analyzer to the engine. It returns an error if an
// analyzer with the same Name is already registered — silently
// overwriting a plugin by name would hide a configuration mistake.
func (e *Engine) Register(a Analyzer) error {
	if a == nil {
		return errors.New("analysis: analyzer must not be nil")
	}
	if a.Name() == "" {
		return errors.New("analysis: analyzer Name must not be empty")
	}
	if _, exists := e.analyzers[a.Name()]; exists {
		return fmt.Errorf("analysis: analyzer %q is already registered", a.Name())
	}
	e.analyzers[a.Name()] = a
	e.order = append(e.order, a.Name())
	return nil
}

// Analyzers returns the registered analyzers in registration order.
func (e *Engine) Analyzers() []Analyzer {
	out := make([]Analyzer, 0, len(e.order))
	for _, name := range e.order {
		out = append(out, e.analyzers[name])
	}
	return out
}

// Run executes every registered analyzer against every Scope it applies
// to within the given Tournament. One analyzer's failure does not prevent
// other analyzers (or other scopes of the same analyzer) from running —
// analyzers are independent, so a defect in one must not silently hide
// the results of the others. All per-(analyzer, scope) errors are joined
// and returned alongside whatever Analyses succeeded.
func (e *Engine) Run(ctx context.Context, tournament *domain.Tournament) ([]domain.Analysis, error) {
	var results []domain.Analysis
	var errs []error

loop:
	for _, name := range e.order {
		analyzer := e.analyzers[name]
		scopes := enumerateScopes(tournament, analyzer.ScopeType())

		for _, scope := range scopes {
			if err := ctx.Err(); err != nil {
				errs = append(errs, err)
				break loop
			}

			hash := sourceHash(tournament, scope, name)
			if cached, ok := e.getCached(hash); ok {
				results = append(results, cached)
				continue
			}

			result, err := analyzer.Analyze(ctx, Input{Tournament: tournament, Scope: scope})
			if err != nil {
				errs = append(errs, fmt.Errorf("analyzer %q (scope %s/%s): %w", name, scope.Type, scope.ID, err))
				continue
			}
			if err := validateFindings(result.Findings); err != nil {
				errs = append(errs, fmt.Errorf("analyzer %q (scope %s/%s) produced invalid result: %w", name, scope.Type, scope.ID, err))
				continue
			}

			analysis := domain.Analysis{
				AnalyzerName: name,
				Scope:        scope,
				SourceHash:   hash,
				GeneratedAt:  e.Now(),
				Score:        result.Score,
				Metrics:      result.Metrics,
				Findings:     result.Findings,
			}
			e.setCached(hash, analysis)
			results = append(results, analysis)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].AnalyzerName != results[j].AnalyzerName {
			return results[i].AnalyzerName < results[j].AnalyzerName
		}
		return results[i].Scope.ID < results[j].Scope.ID
	})

	return results, errors.Join(errs...)
}

// getCached returns a cached Analysis for hash, along with a defensive
// copy of its Metrics map and Findings slice so a caller mutating the
// returned value can never corrupt the cache entry (or another caller's
// previously-returned copy) through it.
func (e *Engine) getCached(hash string) (domain.Analysis, bool) {
	e.cacheMu.RLock()
	a, ok := e.cache[hash]
	e.cacheMu.RUnlock()
	if !ok {
		return domain.Analysis{}, false
	}
	return cloneAnalysis(a), true
}

// setCached stores a defensive copy of a in the cache, keyed by hash.
func (e *Engine) setCached(hash string, a domain.Analysis) {
	e.cacheMu.Lock()
	e.cache[hash] = cloneAnalysis(a)
	e.cacheMu.Unlock()
}

func cloneAnalysis(a domain.Analysis) domain.Analysis {
	clone := a
	if a.Metrics != nil {
		clone.Metrics = make(map[string]float64, len(a.Metrics))
		for k, v := range a.Metrics {
			clone.Metrics[k] = v
		}
	}
	if a.Findings != nil {
		clone.Findings = make([]domain.Finding, len(a.Findings))
		for i, f := range a.Findings {
			if f.Metrics != nil {
				m := make(map[string]float64, len(f.Metrics))
				for k, v := range f.Metrics {
					m[k] = v
				}
				f.Metrics = m
			}
			clone.Findings[i] = f
		}
	}
	return clone
}

// validateFindings enforces docs/06-domain-model.md's domain rule that
// validateFindings reports an error if any finding has an invalid severity or is missing a reason or recommendation.
// It returns the first validation error encountered and includes the finding index in the error message.
func validateFindings(findings []domain.Finding) error {
	for i, f := range findings {
		switch f.Severity {
		case domain.SeverityInfo, domain.SeverityWarning, domain.SeverityCritical:
		default:
			return fmt.Errorf("finding[%d]: invalid or missing Severity %q", i, f.Severity)
		}
		if f.Reason == "" {
			return fmt.Errorf("finding[%d]: Reason is required", i)
		}
		if f.Recommendation == "" {
			return fmt.Errorf("finding[%d]: Recommendation is required", i)
		}
	}
	return nil
}
