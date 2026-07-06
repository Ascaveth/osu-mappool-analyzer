package modmap

import "testing"

func approxEqual(a, b float64) bool {
	const epsilon = 0.01
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < epsilon
}

func TestEffectiveDifficultyFor_NoMod(t *testing.T) {
	eff := EffectiveDifficultyFor(9, 8, 4, 5, 180, 120, NoMod)
	if eff.AR != 9 || eff.OD != 8 || eff.CS != 4 || eff.HP != 5 || eff.BPM != 180 || eff.LengthSeconds != 120 {
		t.Errorf("NoMod changed values: %+v", eff)
	}
}

func TestEffectiveDifficultyFor_HardRock(t *testing.T) {
	eff := EffectiveDifficultyFor(9, 8, 4, 5, 180, 120, ModHardRock)
	if !approxEqual(eff.CS, 5.2) { // 4 * 1.3
		t.Errorf("CS = %v, want ~5.2", eff.CS)
	}
	if !approxEqual(eff.AR, 10) { // min(9*1.4, 10) = 12.6 -> clamped to 10
		t.Errorf("AR = %v, want 10 (clamped)", eff.AR)
	}
	if !approxEqual(eff.OD, 10) { // min(8*1.4, 10) = 11.2 -> clamped
		t.Errorf("OD = %v, want 10 (clamped)", eff.OD)
	}
	if !approxEqual(eff.HP, 7) { // 5 * 1.4
		t.Errorf("HP = %v, want 7", eff.HP)
	}
	if eff.BPM != 180 {
		t.Errorf("BPM = %v, want unchanged 180 (HR doesn't affect rate)", eff.BPM)
	}
}

func TestEffectiveDifficultyFor_HardRock_DoesNotClampBelowTen(t *testing.T) {
	eff := EffectiveDifficultyFor(5, 5, 5, 5, 180, 120, ModHardRock)
	if !approxEqual(eff.AR, 7) { // 5 * 1.4
		t.Errorf("AR = %v, want 7", eff.AR)
	}
	if !approxEqual(eff.CS, 6.5) { // 5 * 1.3
		t.Errorf("CS = %v, want 6.5", eff.CS)
	}
}

func TestEffectiveDifficultyFor_DoubleTime_ScalesBPMAndLength(t *testing.T) {
	eff := EffectiveDifficultyFor(9, 8, 4, 5, 180, 120, ModDoubleTime)
	if !approxEqual(eff.BPM, 270) { // 180 * 1.5
		t.Errorf("BPM = %v, want 270", eff.BPM)
	}
	if eff.LengthSeconds != 80 { // 120 / 1.5
		t.Errorf("LengthSeconds = %v, want 80", eff.LengthSeconds)
	}
	if eff.CS != 4 {
		t.Errorf("CS = %v, want unchanged 4 (DT doesn't affect CS)", eff.CS)
	}
	// DT raises effective AR/OD by compressing their timing windows, but
	// does not scale them by a fixed multiplier the way HR does.
	if eff.AR <= 9 {
		t.Errorf("AR = %v, want > 9 (DT should raise effective AR)", eff.AR)
	}
	if eff.OD <= 8 {
		t.Errorf("OD = %v, want > 8 (DT should raise effective OD)", eff.OD)
	}
}

func TestEffectiveDifficultyFor_HalfTime_LowersEffectiveARAndOD(t *testing.T) {
	eff := EffectiveDifficultyFor(9, 8, 4, 5, 180, 120, ModHalfTime)
	if !approxEqual(eff.BPM, 135) { // 180 * 0.75
		t.Errorf("BPM = %v, want 135", eff.BPM)
	}
	if eff.AR >= 9 {
		t.Errorf("AR = %v, want < 9 (HT should lower effective AR)", eff.AR)
	}
	if eff.OD >= 8 {
		t.Errorf("OD = %v, want < 8 (HT should lower effective OD)", eff.OD)
	}
}

func TestEffectiveDifficultyFor_DTHR_StacksBothEffects(t *testing.T) {
	eff := EffectiveDifficultyFor(9, 8, 4, 5, 180, 120, ModDoubleTime|ModHardRock)
	if !approxEqual(eff.BPM, 270) {
		t.Errorf("BPM = %v, want 270", eff.BPM)
	}
	if !approxEqual(eff.CS, 5.2) {
		t.Errorf("CS = %v, want 5.2 (HR's CS multiplier still applies)", eff.CS)
	}
	if eff.AR != 10 {
		t.Errorf("AR = %v, want 10 (HR already clamps AR to max)", eff.AR)
	}
}

func TestEffectiveDifficultyFor_Easy(t *testing.T) {
	eff := EffectiveDifficultyFor(9, 8, 4, 5, 180, 120, ModEasy)
	if !approxEqual(eff.AR, 4.5) || !approxEqual(eff.OD, 4) || !approxEqual(eff.CS, 2) || !approxEqual(eff.HP, 2.5) {
		t.Errorf("Easy scaling wrong: %+v", eff)
	}
}
