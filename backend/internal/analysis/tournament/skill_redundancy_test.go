package tournament

import (
	"math"
	"testing"
)

func TestSimilarityFixedScales(t *testing.T) {
	a := skillProfileVector{200, 10, 32, 4, .5, .2, 140}
	b := a
	b[0] += .1
	base := normalizeVectors([]skillProfileVector{a, b})
	distance := rmsDistance(base[0], base[1])
	if distance <= 0 || distance >= .01 {
		t.Fatalf("small change distance: %v", distance)
	}
	extended := normalizeVectors([]skillProfileVector{a, b, {10000, 1000, 1000, 100, 1, 1, 1000}})
	if rmsDistance(extended[0], extended[1]) != distance {
		t.Fatal("outlier changed pair distance")
	}
	for _, v := range []skillProfileVector{{}, a} {
		n := normalizeVectors([]skillProfileVector{v, v})
		if d := rmsDistance(n[0], n[1]); d != 0 || math.IsNaN(d) || math.IsInf(d, 0) {
			t.Fatalf("identical distance: %v", d)
		}
	}
	if d := rmsDistance(extended[0], extended[2]); d < redundancySimilarityThreshold {
		t.Fatalf("extreme profile similar: %v", d)
	}
}
