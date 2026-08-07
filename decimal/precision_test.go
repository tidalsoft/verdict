package decimal

import "testing"

// TestPrecisionLoss_ConformanceVectors encodes SPEC-MU §8 vectors 9, 10, and
// 11 directly. These are the acceptance tests for MU-02 at this layer.
func TestPrecisionLoss_ConformanceVectors(t *testing.T) {
	cases := []struct {
		vector     int
		name       string
		value      string
		provenance Provenance
		want       bool // want == true means MU-02 would FAIL
	}{
		{9, "money, decimal string -> PASS", "49.99", FromString, false},
		{10, "money, JSON number 0.1 -> FAIL (not exactly representable)", "0.1", FromJSONNumber, true},
		{11, "money, JSON number > 2^53-1 -> FAIL", "9007199254740993", FromJSONNumber, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := mustParse(t, tc.value)
			got := PrecisionLoss(d, tc.provenance)
			if got != tc.want {
				t.Errorf("vector %d: PrecisionLoss(%s, %v) = %v, want %v", tc.vector, tc.value, tc.provenance, got, tc.want)
			}
		})
	}
}

// TestPrecisionLoss_SameValueStringVsJSONNumber is the provenance distinction
// the task calls out explicitly: the identical numeric value must PASS as a
// decimal string and FAIL as a JSON number (vectors 9 vs 10 use different
// values to make the point; this test uses one value both ways to prove the
// distinction is provenance-driven, not value-driven).
func TestPrecisionLoss_SameValueStringVsJSONNumber(t *testing.T) {
	d := mustParse(t, "0.1")

	if PrecisionLoss(d, FromString) {
		t.Error("0.1 as FromString: PrecisionLoss = true, want false (decimal strings are exact by construction)")
	}
	if !PrecisionLoss(d, FromJSONNumber) {
		t.Error("0.1 as FromJSONNumber: PrecisionLoss = false, want true (not exactly representable in binary64)")
	}
}

func TestExactlyRepresentableInBinary64(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"0.5", true},               // exact power-of-two fraction
		{"0.25", true},              // exact power-of-two fraction
		{"0", true},                 // zero is exact
		{"1", true},                 // small integers are exact
		{"0.1", false},              // classic inexact decimal fraction
		{"49.99", false},            // vector-adjacent inexact value
		{"-0.5", true},              // negative exactness mirrors positive
		{"9007199254740992", true},  // 2^53, still exactly representable
		{"9007199254740993", false}, // 2^53+1, first inexact integer
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			d := mustParse(t, tc.value)
			if got := d.ExactlyRepresentableInBinary64(); got != tc.want {
				t.Errorf("ExactlyRepresentableInBinary64(%s) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

// TestExactlyRepresentableInBinary64_ExtremeMagnitudes covers 10^-400 and
// 10^400 -- magnitudes that underflow to 0 and overflow to +Inf in binary64
// respectively, so neither is exact. These are built directly rather than
// via Parse: expressing them as Go string literals would need a 400+ digit
// plain-decimal string, and Parse itself no longer accepts the scientific
// notation ("1e-400") that would otherwise write them concisely (see
// Parse's doc comment).
func TestExactlyRepresentableInBinary64_ExtremeMagnitudes(t *testing.T) {
	tiny := newTestDecimal(1, -400) // 10^-400: underflows to 0, not exact
	if tiny.ExactlyRepresentableInBinary64() {
		t.Error("ExactlyRepresentableInBinary64(1e-400) = true, want false")
	}
	huge := newTestDecimal(1, 400) // 10^400: overflows to +Inf, not exact
	if huge.ExactlyRepresentableInBinary64() {
		t.Error("ExactlyRepresentableInBinary64(1e400) = true, want false")
	}
}

func TestExceedsSafeIntegerMagnitude(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"9007199254740991", false}, // 2^53-1, exactly at the boundary: PASS
		{"-9007199254740991", false},
		{"9007199254740992", true}, // 2^53, one past the boundary
		{"-9007199254740992", true},
		{"9007199254740993", true}, // vector 11
		{"0", false},
		{"49.99", false},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			d := mustParse(t, tc.value)
			if got := d.ExceedsSafeIntegerMagnitude(); got != tc.want {
				t.Errorf("ExceedsSafeIntegerMagnitude(%s) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

// TestPrecisionLoss_VeryLargeAndVerySmallMagnitudes exercises magnitudes far
// outside anything binary64 or the 2^53 ceiling were designed around, per
// the task's required coverage.
func TestPrecisionLoss_VeryLargeAndVerySmallMagnitudes(t *testing.T) {
	tiny := mustParse(t, "0.000000000000000000001")
	if !PrecisionLoss(tiny, FromJSONNumber) {
		t.Error("tiny value as FromJSONNumber: want PrecisionLoss true (not exactly representable)")
	}
	if PrecisionLoss(tiny, FromString) {
		t.Error("tiny value as FromString: want PrecisionLoss false")
	}

	huge := mustParse(t, "99999999999999999999999999999999")
	if !PrecisionLoss(huge, FromJSONNumber) {
		t.Error("huge value as FromJSONNumber: want PrecisionLoss true (exceeds safe integer magnitude)")
	}
}
