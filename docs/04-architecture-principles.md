# Architecture Principles

These principles govern every design and implementation decision in this project. They take precedence over convenience, familiarity, or speed of delivery.

## 1. Analysis-first, not visualization-first

The Analysis Engine is the source of truth. The REST API, reports, and frontend are presentation layers over its output — they never drive design decisions. If a UI need would require distorting the Analysis Engine's model, the UI adapts, not the engine.

## 2. The core pipeline is fixed; everything inside it is pluggable

```
Tournament Data → Normalization Engine → Analysis Engine → Structured Analysis Results → {REST API, Reports, Web UI, Charts}
```

The pipeline shape is stable. The *analyzers* running inside the Analysis Engine stage are not — new analyzers must be addable without touching existing ones.

## 3. Analyzers are independent plugins

- Each analyzer has a single responsibility (one category of insight: metadata, pattern, composition, progression, balance, diversity, validation, etc.).
- Analyzers accept normalized data and return a structured result (findings, metrics, recommendations, severities). They never call each other or share mutable state.
- Adding an analyzer must never require modifying an existing analyzer's code. If it does, the analyzer interface is wrong and needs revisiting before more analyzers are added.

## 4. Tournament structure is always user-defined

No stage name, mod category, slot count, or ordering is ever hardcoded. The system must adapt to a tournament's configuration, not the reverse. Treat "Qualifiers / RO16 / Quarterfinals / Finals" as an *example*, never an assumption baked into code or schema.

## 5. Data philosophy

- **Beatmaps are immutable source data.** Once imported, raw beatmap data is not mutated by analysis.
- **Tournament configuration is user-defined rules**, stored explicitly, never inferred.
- **Analysis results are derived data.** They must always be fully regenerable from source data + configuration, and should not be treated as a second source of truth. Avoid persisting derived data that can't be reproduced deterministically.

## 6. Determinism

Given the same beatmaps and the same tournament configuration, an analyzer must always produce the same result. No analyzer should depend on wall-clock time, randomness, or external network state for its conclusions.

## 7. Explicit over clever

Favor straightforward, readable code over abstractions that save lines but cost comprehension. A future contributor (including future-us) should be able to read an analyzer and understand its logic without reverse-engineering intent.

## 8. Technology agnosticism

No technology is assumed by default (notably: do not assume Java/Spring Boot out of habit). Each technology choice — backend language/framework, database, cache, deployment — is evaluated against the requirements of the specific problem it solves, with trade-offs stated explicitly. See the stack proposal in [docs/05-stack-proposal.md](05-stack-proposal.md) once that decision point is reached.

## 9. Reports speak in conclusions, not raw numbers

A report says *"the Finals stage introduces a noticeable increase in technical difficulty while maintaining consistent BPM progression,"* not *"average BPM increased by 12."* Every finding surfaced to a user must explain why it matters, not just what was measured. This is an architectural constraint, not a copywriting concern — it means findings must carry semantic context (what was compared, what threshold/expectation was used, why that threshold is meaningful), not just a number.

## 10. SOLID, Clean Architecture, composition over inheritance

- Single Responsibility per analyzer, per module, per layer.
- Dependency direction flows inward: outer layers (API, UI) depend on the Analysis Engine's interfaces; the engine never depends on presentation concerns.
- Prefer composing small, focused units over deep inheritance hierarchies.

## 11. Testability is a design constraint, not an afterthought

Every analyzer must be testable in isolation against synthetic normalized data, without needing a real tournament, a real database, or other analyzers. If an analyzer can't be unit-tested without standing up infrastructure, its boundaries are wrong.
