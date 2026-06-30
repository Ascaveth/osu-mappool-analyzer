package domain

// Tournament is the root of the configuration aggregate described in
// docs/06-domain-model.md: Tournament owns Stage owns Category owns Slot,
// edited and read together as one unit. Slots reference Beatmap by value
// here (post-load, for analyzer convenience) but Beatmap remains its own
// aggregate — nothing in this file mutates a Beatmap.
type Tournament struct {
	ID      string
	Name    string
	Edition string
	Stages  []Stage
}

// Stage is one round of a Tournament (Qualifiers, RO16, Finals, ...).
// Order is explicit and authoritative for progression analysis — stage
// sequence is never inferred from name or slice position.
type Stage struct {
	ID         string
	Name       string
	Order      int
	Categories []Category
}

// Category is a mod/intent grouping within a Stage (NM, HD, HR, ...).
// Category.Name is free text per docs/04 Architecture Principle 4 — never
// validated against a fixed enum.
type Category struct {
	ID    string
	Name  string
	Order int
	Slots []Slot
}

// Slot is a single beatmap position within a Category. Beatmap is nil when
// the slot has not been filled yet — pools are built incrementally, and
// analyzers must treat an unfilled slot as a finding opportunity, not an
// error condition (docs/06-domain-model.md domain rules).
type Slot struct {
	ID       string
	Position int
	Beatmap  *Beatmap
}
