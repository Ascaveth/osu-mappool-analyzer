package modmap

// Mod-scaling constants from osu!'s classic (stable) ruleset. HR/EZ scale
// CS/AR/OD/HP by a fixed multiplier (CS gets its own, slightly gentler,
// multiplier than AR/OD/HP); DT/HT instead change playback rate, which
// only affects the two timing-derived settings (AR, OD) plus BPM/length —
// CS and HP are spatial, not timing, so a rate change leaves them alone.
const (
	hrCSMultiplier    = 1.3
	hrOtherMultiplier = 1.4
	ezMultiplier      = 0.5

	doubleTimeRate = 1.5
	halfTimeRate   = 0.75
)

// AR/OD <-> milliseconds conversions, needed to rescale the *timing*
// settings under a DT/HT rate change (rather than the AR/OD numbers
// themselves, which are not linear in time). These are osu!'s documented
// game-client constants, not tournament convention.
const (
	arLowBreakpoint = 5.0
	odBaseWindowMs  = 79.5
	odPerPointMs    = 6.0
)

func approachTimeMs(ar float64) float64 {
	if ar < arLowBreakpoint {
		return 1800 - 120*ar
	}
	return 1200 - 150*(ar-arLowBreakpoint)
}

func arFromApproachTimeMs(ms float64) float64 {
	if ms > 1200 {
		return (1800 - ms) / 120
	}
	return arLowBreakpoint + (1200-ms)/150
}

func hitWindowMs(od float64) float64 {
	return odBaseWindowMs - odPerPointMs*od
}

func odFromHitWindowMs(ms float64) float64 {
	return (odBaseWindowMs - ms) / odPerPointMs
}

func clamp0to10(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 10:
		return 10
	default:
		return v
	}
}

// EffectiveDifficulty is a beatmap's AR/OD/CS/HP/BPM/length as they
// actually play under a fixed set of Mods, as opposed to the raw values
// read from the .osu file. Unlike Star Rating — which requires osu!'s full
// difficulty-calculation algorithm and is fetched from the osu! API at
// import time (see internal/enrich) rather than recomputed locally — these
// are deterministic arithmetic transforms defined by the mods themselves,
// safe to compute wherever a mod-specific view of metadata is needed.
type EffectiveDifficulty struct {
	AR, OD, CS, HP float64
	BPM            float64
	LengthSeconds  int
}

// EffectiveDifficultyFor applies m's known Star-Rating-affecting scaling
// (HR, EZ, DT, HT) to a beatmap's raw AR/OD/CS/HP/BPM/length, in the same
// order osu! itself applies them: HR/EZ's fixed CS/AR/OD/HP multipliers
// first, then DT/HT's playback-rate change (which only touches the two
// timing settings, AR and OD, plus BPM and length) — so a combo like DTHR
// scales AR/OD/CS/HP for HR and then further rescales AR/OD's timing for
// DT, matching how the two mods actually stack in-game.
//
// Hidden and Flashlight are not handled here — neither changes any of
// these values (see AffectsStarRating).
func EffectiveDifficultyFor(ar, od, cs, hp, bpm float64, lengthSeconds int, m Mods) EffectiveDifficulty {
	if m&ModHardRock != 0 {
		cs = clamp0to10(cs * hrCSMultiplier)
		ar = clamp0to10(ar * hrOtherMultiplier)
		od = clamp0to10(od * hrOtherMultiplier)
		hp = clamp0to10(hp * hrOtherMultiplier)
	}
	if m&ModEasy != 0 {
		cs = clamp0to10(cs * ezMultiplier)
		ar = clamp0to10(ar * ezMultiplier)
		od = clamp0to10(od * ezMultiplier)
		hp = clamp0to10(hp * ezMultiplier)
	}

	rate := 1.0
	switch {
	case m&ModDoubleTime != 0:
		rate = doubleTimeRate
	case m&ModHalfTime != 0:
		rate = halfTimeRate
	}

	if rate != 1.0 {
		ar = clamp0to10(arFromApproachTimeMs(approachTimeMs(ar) / rate))
		od = clamp0to10(odFromHitWindowMs(hitWindowMs(od) / rate))
		bpm *= rate
		lengthSeconds = int(float64(lengthSeconds) / rate)
	}

	return EffectiveDifficulty{AR: ar, OD: od, CS: cs, HP: hp, BPM: bpm, LengthSeconds: lengthSeconds}
}
