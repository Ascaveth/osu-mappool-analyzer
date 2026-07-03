# Tournament Configuration Specification

Phase 3 deliverable: how a user expresses an arbitrary tournament's structure as data, before any beatmap is assigned to a slot. This specification governs the input that creates the `Tournament` aggregate defined in [06-domain-model.md](06-domain-model.md).

## Design goal

A tournament organizer must be able to describe *any* stage/category/slot structure — one stage or ten, three categories or twelve, two slots in a category or eight — without the system assuming anything about names, counts, or ordering. The configuration format is the single mechanism through which [Architecture Principle 4](04-architecture-principles.md#4-tournament-structure-is-always-user-defined) is enforced at the boundary: if the format can't express it, no later layer should try to special-case it back in.

## Configuration shape

A tournament configuration is a `Tournament` plus an ordered list of `Stage`s, each with an ordered list of `Category`s, each declaring a slot count. Slots themselves are *generated* from `slotCount`, not hand-enumerated — a slot doesn't need configuration data of its own (a name or rule), only a position, which is assigned sequentially.

```json
{
  "name": "Example Open",
  "edition": "2026",
  "stages": [
    {
      "name": "Qualifiers",
      "order": 1,
      "categories": [
        { "name": "NM", "order": 1, "slotCount": 5 },
        { "name": "HD", "order": 2, "slotCount": 2 },
        { "name": "HR", "order": 3, "slotCount": 2 },
        { "name": "DT", "order": 4, "slotCount": 2 },
        { "name": "FM", "order": 5, "slotCount": 2 }
      ]
    },
    {
      "name": "Round of 16",
      "order": 2,
      "categories": [
        { "name": "NM", "order": 1, "slotCount": 4 },
        { "name": "HD", "order": 2, "slotCount": 1 },
        { "name": "HR", "order": 3, "slotCount": 1 },
        { "name": "DT", "order": 4, "slotCount": 1 },
        { "name": "FM", "order": 5, "slotCount": 2 },
        { "name": "TB", "order": 6, "slotCount": 1 }
      ]
    },
    {
      "name": "Grand Finals",
      "order": 3,
      "categories": [
        { "name": "NM", "order": 1, "slotCount": 6 },
        { "name": "HD", "order": 2, "slotCount": 3 },
        { "name": "HR", "order": 3, "slotCount": 2 },
        { "name": "DT", "order": 4, "slotCount": 3 },
        { "name": "FM", "order": 5, "slotCount": 2 },
        { "name": "TB", "order": 6, "slotCount": 1 }
      ]
    }
  ]
}
```

Applying this configuration creates the `Tournament` aggregate: one row in `tournament`, N rows in `stage`, M rows in `category`, and `slotCount` rows in `slot` per category — all with `beatmap_id = NULL`. Filling slots with beatmaps is a separate, later operation (Phase 4 import + a slot-assignment step), not part of configuration.

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "TournamentConfiguration",
  "type": "object",
  "required": ["name", "stages"],
  "properties": {
    "name": { "type": "string", "minLength": 1 },
    "edition": { "type": "string" },
    "stages": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#/definitions/stage" }
    }
  },
  "definitions": {
    "stage": {
      "type": "object",
      "required": ["name", "order", "categories"],
      "properties": {
        "name": { "type": "string", "minLength": 1 },
        "order": { "type": "integer", "minimum": 1 },
        "categories": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/definitions/category" }
        }
      }
    },
    "category": {
      "type": "object",
      "required": ["name", "order", "slotCount"],
      "properties": {
        "name": { "type": "string", "minLength": 1 },
        "order": { "type": "integer", "minimum": 1 },
        "slotCount": { "type": "integer", "minimum": 1 }
      }
    }
  }
}
```

This schema is the contract the REST API (Phase 10) validates against on tournament creation/update. It is deliberately minimal — it describes shape, not domain rules that require cross-field checks (those are listed below and enforced in application code, since JSON Schema alone can't express "orders must be unique within a stage").

## Validation rules

Beyond the JSON Schema's shape checks:

1. **`Stage.order` is *not* required to be unique within a Tournament.** Two stages may share a position deliberately — see "Supporting future/non-linear formats" below, where same-`order` stages express a parallel/concurrent peer set (e.g. simultaneous group pools). Progression analysis treats a shared order as "no strict sequence step between these," not as malformed input.
2. **`Category.order` is unique within a Stage.** Same reasoning, one level down.
3. **`Category.name` is not required to be unique within a Stage**, but a duplicate name within the same stage produces a configuration warning (not a hard rejection) — it's almost always a mistake (e.g. two "NM" categories) but isn't formally invalid, since nothing downstream depends on category names being distinct.
4. **`slotCount` must be ≥ 1.** A category with zero slots isn't a category — omit it instead.
5. **No structural limit on counts.** No maximum number of stages, categories, or slots is enforced by the schema — only sane validation-engine *warnings* later (e.g. "this category has an unusually high slot count") are allowed to flag unusual configurations, and warnings never block creation.

## Updating a configuration after creation

A configuration can change after a `Tournament` is created (stages added, slot counts changed, etc.) — pools are built incrementally, not declared once and frozen. Two consequences follow directly from [Architecture Principle 5](04-architecture-principles.md#5-data-philosophy) and the `source_hash` mechanism on `Analysis`:

- Changing a `Stage`, `Category`, or `Slot` count changes the `source_hash` for any `Analysis` scoped to that Tournament or its descendants.
- Existing `Analysis` rows are never mutated to reflect a config change — they remain a valid historical record of "what the analysis said when the config looked like X." A new `Analysis` must be generated against the new configuration. The system never silently presents a stale Analysis as current; callers compare `source_hash` to know whether a re-run is needed.

Removing a `Slot` that currently references a `Beatmap` requires explicit confirmation at the API layer (Phase 10) — it's a destructive edit to in-progress pool-building work, not a pure configuration change.

## Supporting future/non-linear formats

The plan's roadmap calls for the format to remain open to "future tournament formats." This spec deliberately does not model bracket logic (single elim, double elim, group stage, Swiss, etc.) — that's tournament *management*, an explicit non-goal (see [02-scope.md](02-scope.md)). Pool analysis only needs to know the *sequence* of pools a tournament uses, not how players move between them. Consequences:

- A double-elimination bracket with a separate Losers' pool is just two more `Stage` entries (e.g. "Losers Round 1", order 4) — no schema change needed.
- A format with parallel/concurrent stages (e.g. simultaneous group pools) is expressed by giving them the same `order` — progression analysis treats same-order stages as a peer set rather than a strict sequence step, not as an error.
- If a future format needs data this schema can't express, that's a signal to add an optional field to `stage` or `category` (e.g. a free-form `rules` string already implied by the domain model's "rules" concept), not to redesign the shape. The schema's `additionalProperties` is intentionally left unrestricted at the stage/category level for forward-compatible metadata, validated loosely until a real use case defines it formally.

## What this spec does not cover

- **Slot-to-beatmap assignment** — covered by Phase 4 (Import Pipeline) and a subsequent assignment operation, not configuration.
- **Mod category semantics** (what "HD" *means* to an analyzer) at the domain layer — `internal/domain` and most analyzers interpret category names as plain labels, per Architecture Principle 4. One documented, opt-in exception exists: `internal/modmap` is a named convention table (outside `internal/domain`, same pattern as `tournament.DefaultTaxonomy()`) that maps category-name conventions like `"HD"`/`"HR"`/`"FM"` to osu! mod bitflags, used specifically by Star Rating enrichment and `tournament.DifficultySpreadAnalyzer` — see [docs/18-configuration-and-modmap.md](18-configuration-and-modmap.md). This doesn't relax the rule above: every other analyzer still treats `Category.Name` as an opaque label, and `modmap`'s table is overridable/revisitable convention, not a hardcoded domain assumption.
