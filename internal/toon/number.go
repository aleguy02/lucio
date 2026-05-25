package toon

import (
	"math"
	"strconv"
)

// formatFloat formats a finite float64 in TOON canonical form (spec §2).
// Non-finite values are normalized to "null" per §3.
func formatFloat(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "null"
	}
	if v == 0 {
		// -0 normalizes to 0.
		return "0"
	}
	abs := math.Abs(v)
	if abs >= 1e-6 && abs < 1e21 {
		// 'f' with -1 precision yields the shortest round-tripping
		// decimal; e.g. 1.0 → "1", 1.5 → "1.5", 1e-6 → "0.000001".
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	// Exponent form with lowercase 'e' and explicit sign (§2).
	return strconv.FormatFloat(v, 'e', -1, 64)
}

// formatInt formats an int64 (no canonicalization is needed beyond
// strconv's base-10 output).
func formatInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

func formatUint(v uint64) string {
	return strconv.FormatUint(v, 10)
}
