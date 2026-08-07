package decimal

import "testing"

// TestReconciles_ConformanceVectors encodes SPEC-MU §8 vectors 32, 33, and
// 34 directly against this package's exact arithmetic. Vector 33 is called
// out in SPEC-MU §8 as "the single most important test in this document":
// line items [0.1, 0.2] must reconcile to a total of 0.3 under exact decimal
// arithmetic, where binary floating point would fail it.
func TestReconciles_ConformanceVectors(t *testing.T) {
	cases := []struct {
		vector int
		name   string
		items  []string
		total  string
		want   bool
	}{
		{32, "items [10.10, 20.20], total 30.30 -> PASS", []string{"10.10", "20.20"}, "30.30", true},
		{33, "items [0.1, 0.2], total 0.3 -> PASS (exact decimal)", []string{"0.1", "0.2"}, "0.3", true},
		{34, "items [10.10, 20.20], total 30.31 -> FAIL", []string{"10.10", "20.20"}, "30.31", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := make([]Decimal, len(tc.items))
			for i, v := range tc.items {
				items[i] = mustParse(t, v)
			}
			computed, err := Sum(items...)
			if err != nil {
				t.Fatalf("Sum unexpected error: %v", err)
			}
			total := mustParse(t, tc.total)
			zero := mustParse(t, "0")

			got, err := Reconciles(computed, total, zero)
			if err != nil {
				t.Fatalf("Reconciles unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("vector %d: Reconciles(sum=%s, total=%s, tolerance=0) = %v, want %v",
					tc.vector, computed, total, got, tc.want)
			}
		})
	}
}

// TestVector33_ExactDecimalVersusFloat64 is the tripwire test called for
// explicitly by the task: it proves vector 33 passes under this package's
// exact decimal arithmetic, and independently demonstrates that the same
// computation performed in native float64 does NOT reconcile -- so a future
// change that quietly swapped this package's internals for float64 math
// would make the first assertion fail, not just the second.
func TestVector33_ExactDecimalVersusFloat64(t *testing.T) {
	// The float64 half: 0.1 + 0.2 != 0.3 in IEEE 754 binary64. This is not
	// testing this package at all -- it is documenting, in an executable and
	// permanently-checked form, exactly the defect SPEC-MU MU-12 and
	// CLAUDE.md invariant #2 exist to keep out of this codebase.
	//
	// The two operands are assigned to float64 variables first, deliberately:
	// `0.1 + 0.2` written directly as a constant expression would be folded
	// by the Go compiler using arbitrary-precision constant arithmetic and
	// rounded to float64 only once, at the very end -- which equals 0.3
	// exactly and would silently defeat the point of this test. Routing
	// through variables forces two independent, lossy runtime float64
	// roundings, which is what actually happens when a JSON number decodes
	// into a float64 field.
	var floatTenth = 0.1
	var floatTwoTenths = 0.2
	floatSum := floatTenth + floatTwoTenths
	if floatSum == 0.3 {
		t.Fatal("0.1 + 0.2 == 0.3 under runtime float64 on this platform -- the premise of vector 33 " +
			"(and of this package's reason to exist) no longer holds; investigate before trusting " +
			"the exact-decimal assertion below")
	}

	// The exact-decimal half: this package must reconcile the same inputs.
	// If this package's Add/Sum were ever reimplemented on top of float64,
	// this assertion is the one that would start failing.
	tenth := mustParse(t, "0.1")
	twoTenths := mustParse(t, "0.2")
	total := mustParse(t, "0.3")
	zero := mustParse(t, "0")

	sum, err := Sum(tenth, twoTenths)
	if err != nil {
		t.Fatalf("Sum unexpected error: %v", err)
	}
	if sum.Compare(total) != 0 {
		t.Fatalf("exact decimal 0.1 + 0.2 = %s, want exactly %s", sum, total)
	}
	ok, err := Reconciles(sum, total, zero)
	if err != nil {
		t.Fatalf("Reconciles unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("Reconciles(0.1+0.2, 0.3, tolerance=0) = false, want true")
	}
}

func TestSum_Empty(t *testing.T) {
	sum, err := Sum()
	if err != nil {
		t.Fatalf("Sum() unexpected error: %v", err)
	}
	if !sum.IsZero() {
		t.Errorf("Sum() = %s, want 0", sum)
	}
}

func TestSum_PropagatesAddError(t *testing.T) {
	huge := newTestDecimal(9, 100000)
	_, err := Sum(huge, huge)
	if err == nil {
		t.Fatal("Sum of overflowing values succeeded, want error")
	}
}

func TestReconciles_ToleranceBoundaries(t *testing.T) {
	computed := mustParse(t, "30.30")

	cases := []struct {
		name      string
		total     string
		tolerance string
		want      bool
	}{
		{"exact match, zero tolerance", "30.30", "0", true},
		{"delta equals tolerance exactly -> within bound", "30.31", "0.01", true},
		{"delta one minor unit over tolerance -> fails", "30.32", "0.01", false},
		{"negative delta within tolerance (total higher than computed)", "30.29", "0.01", true},
		{"delta exceeds tolerance, negative direction", "30.28", "0.01", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			total := mustParse(t, tc.total)
			tolerance := mustParse(t, tc.tolerance)
			got, err := Reconciles(computed, total, tolerance)
			if err != nil {
				t.Fatalf("Reconciles unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Reconciles(%s, %s, tolerance=%s) = %v, want %v",
					computed, tc.total, tc.tolerance, got, tc.want)
			}
		})
	}
}

func TestReconciles_PropagatesSubError(t *testing.T) {
	huge := newTestDecimal(9, 100000)
	negHuge := newTestDecimal(-9, 100000)
	zero := mustParse(t, "0")

	_, err := Reconciles(huge, negHuge, zero)
	if err == nil {
		t.Fatal("Reconciles over overflowing values succeeded, want error")
	}
}
