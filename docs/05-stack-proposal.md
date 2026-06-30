# Technology Stack Proposal

This is a recommendation, not a commitment. Phase 1 is documentation-only; this exists so the choice is reasoned about before Phase 4+ (Import Pipeline, Analysis Engine) needs an answer.

## Requirements driving the choice

1. **Heavy parsing + numeric/geometric computation** — `.osu` file parsing, pattern analysis (jump distances, angles, stream detection) is CPU-bound work over hit-object sequences.
2. **Plugin-style analyzer architecture** — needs a language with strong interfaces/traits and easy composition, not deep inheritance.
3. **Long-lived, evolving schema** — tournament configs, beatmaps, and analysis results need a relational model that can grow without painful migrations.
4. **Deterministic, testable units** — analyzers should be trivially unit-testable in isolation.
5. **Small team / solo-friendly** — avoid stacks that need a platform team to operate.

## Backend: Go

| Option | Verdict |
|---|---|
| **Go** | **Recommended.** Fast enough for parsing/pattern math, simple concurrency for batch beatmap processing, interfaces are a natural fit for the analyzer-plugin contract, single static binary simplifies deployment, low operational overhead for a solo/small team. |
| Rust | Best raw performance and correctness guarantees, but higher development friction for an evolving domain model that will be reshaped often in early phases. Worth revisiting if pattern analysis becomes a real bottleneck. |
| NestJS (TypeScript) | Good DX and shares a language with the frontend, but weaker fit for CPU-heavy parsing/geometry work and its DI-heavy style encourages exactly the kind of implicit coupling the plugin architecture wants to avoid. |
| Spring Boot | Explicitly de-prioritized per project default; no requirement here justifies its operational weight. |
| ASP.NET | Comparable to Go technically, but no team familiarity advantage stated and smaller ecosystem fit for this domain (osu! tooling ecosystem skews Go/Rust/Node). |

**Analyzer interface fit:** Go's small interfaces (`Analyze(ctx, NormalizedPool) (AnalysisResult, error)`) map directly onto the plugin requirement in principle 3 — each analyzer is a package implementing one interface, registered into the engine, with zero compile-time dependency on other analyzers.

## Database: PostgreSQL

No real alternative considered — relational integrity (tournaments → stages → categories → slots → beatmaps) is exactly what the domain needs, JSONB covers the parts of beatmap metadata that don't deserve full normalization (e.g. raw timing point arrays), and it's the project's own listed default.

## Cache: Redis (deferred)

Not needed until there's a real read-heavy API workload (e.g. repeated report regeneration, comparison queries). Don't introduce it in Phase 4/5; revisit once the REST API (Phase 10) has real traffic patterns to optimize for.

## Frontend: Next.js + TypeScript + TailwindCSS + shadcn/ui

Deferred decision in practice (Phase 11), but no reason to deviate from the project's own default list — it's the conventional, well-supported choice for a data-presentation frontend and nothing about this domain pushes toward an alternative.

## Deployment: Docker + Docker Compose

Sufficient for the project's current scale (no indication of needing orchestration like Kubernetes). Keep it simple until evidence says otherwise.

## Summary

| Layer | Choice | Confidence |
|---|---|---|
| Backend | Go | High |
| Database | PostgreSQL | High |
| Cache | Redis | Deferred until needed |
| Frontend | Next.js / TS / Tailwind / shadcn | Medium (revisit at Phase 11) |
| Deployment | Docker Compose | High for current scale |

## Trade-offs accepted

- Go's lack of generics-heavy ergonomics (pre-1.18 pain is gone, but it's still less expressive than TS/Rust for complex type modeling) is accepted in exchange for simplicity and performance.
- Choosing Go over Rust trades a small amount of raw performance for significantly faster iteration speed during the domain-model-heavy early phases (2–8), where the model will change often.

## Open question for the user

Confirm or override this before Phase 4 (Import Pipeline) begins, since that's the first phase that produces real backend code.
