// Package modmap translates the tournament pool model's free-text
// Category.Name convention (docs/04-architecture-principles.md, Principle
// 4: Category.Name is never validated against a fixed enum) into osu!
// Mod bitflags. This is a named, overridable convention table — not a
// domain rule — so it lives outside internal/domain, mirroring how
// tournament.DefaultTaxonomy() keeps its skillset conventions outside
// domain too.
package modmap

import "strings"

// Mods is a bitflag set of osu! mods that affect Star Rating. It is
// defined here (not in internal/osuapi) so that internal/analysis/tournament
// can depend on it without depending on the I/O-concerned osuapi package —
// osuapi depends on modmap, not the reverse.
type Mods uint32

// NoMod represents no active mod (nomod Star Rating).
const NoMod Mods = 0

// Explicit bit assignments, so values stay stable and self-documenting
// regardless of declaration order.
const (
	ModHardRock   Mods = 1 << 0
	ModDoubleTime Mods = 1 << 1
	ModEasy       Mods = 1 << 2
	ModHalfTime   Mods = 1 << 3
	ModHidden     Mods = 1 << 4 // does not affect Star Rating; kept for combo naming only
	ModFlashlight Mods = 1 << 5
)

// srAffectingTable maps category-name convention tokens to the Mods that
// actually change Star Rating under osu!'s classic (stable) difficulty
// algorithm:
//   - HR (Hard Rock): scales CS x1.3, AR/OD/HP x1.4 (capped at 10) - raises SR.
//   - DT (Double Time): 1.5x playback speed, recalculates effective AR/OD -
//     raises SR.
//   - EZ (Easy): roughly halves CS/AR/OD/HP - lowers SR.
//   - HT (Half Time): 0.75x playback speed - lowers SR.
//   - FL (Flashlight): no timing/pattern change - does not affect SR.
//   - HD (Hidden): visibility only, no timing/pattern change - does not
//     affect SR. Recognized here only so combo names like "HDHR" decompose
//     correctly; HD alone never yields a distinct SR value to fetch.
var srAffectingTable = map[string]Mods{
	"HR": ModHardRock,
	"DT": ModDoubleTime,
	"EZ": ModEasy,
	"HT": ModHalfTime,
	"HD": ModHidden,
	"FL": ModFlashlight,
}

// noFixedModNames marks category-name conventions with no single fixed mod
// choice at pool-build time. FreeMod (FM): players choose their own mods
// per play, in practice drawn from {NoMod, HardRock, Easy} — Double Time is
// not a legal FreeMod pick in this convention (see FreeModCandidates).
// Tiebreaker (TB) conventionally carries no fixed mod either.
var noFixedModNames = map[string]bool{
	"FM": true,
	"TB": true,
}

// FreeModCandidates lists the mods a FreeMod ("FM") slot's difficulty
// range should be computed across. Deliberately excludes DoubleTime — DT
// is not a legal FreeMod pick under this project's tournament convention,
// even though it changes SR the same way HR does.
var FreeModCandidates = []Mods{NoMod, ModHardRock, ModEasy}

// FromCategoryName resolves a Category.Name convention string to the Mods
// that change Star Rating for that category. It returns (0, false) when
// the name has no single fixed mod (FreeMod, Tiebreaker, or any
// unrecognized name) — callers must not treat (0, false) as NoMod, since
// "no fixed mod" and "explicitly NoMod" are different states (see
// IsFreeMod to distinguish FreeMod from a genuinely unresolvable name).
//
// Combo names (e.g. "HDHR", "DTHD") are resolved by decomposing the name
// into known two-letter tokens and OR-ing their Mods together; HD
// contributes to the combo's identity but never to its own distinct SR
// value. A name that doesn't fully decompose into known tokens is treated
// as unresolvable, same as an unrecognized single name.
func FromCategoryName(name string) (Mods, bool) {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" {
		return 0, false
	}
	if upper == "NM" {
		return NoMod, true
	}
	if noFixedModNames[upper] {
		return 0, false
	}
	if mods, ok := srAffectingTable[upper]; ok {
		return mods, true
	}

	// Combo decomposition: consume the string two characters at a time,
	// requiring every token to be a known mod. This is a heuristic
	// convention, not authoritative tournament rule parsing (mirrors
	// tournament.jumpDistanceThreshold's documented-heuristic precedent).
	if len(upper)%2 != 0 {
		return 0, false
	}
	var combo Mods
	for i := 0; i < len(upper); i += 2 {
		token := upper[i : i+2]
		mods, ok := srAffectingTable[token]
		if !ok {
			return 0, false
		}
		combo |= mods
	}
	return combo, true
}

// IsFreeMod reports whether name is the FreeMod convention ("FM"),
// distinguishing it from other unresolvable names (e.g. "TB", typos) so
// callers can apply FreeModCandidates ranging instead of skipping outright.
func IsFreeMod(name string) bool {
	return strings.ToUpper(strings.TrimSpace(name)) == "FM"
}

// AffectsStarRating reports whether m includes any mod that changes Star
// Rating under osu!'s classic algorithm (i.e. m is not just Hidden/NoMod).
func AffectsStarRating(m Mods) bool {
	return m&(ModHardRock|ModDoubleTime|ModEasy|ModHalfTime) != 0
}
