package tables

import (
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
	"github.com/tidalsoft/verdict/decimal"
)

func TestDimension_String(t *testing.T) {
	cases := []struct {
		name string
		d    Dimension
		want string
	}{
		{"mass", DimensionMass, "mass"},
		{"length", DimensionLength, "length"},
		{"volume", DimensionVolume, "volume"},
		{"time", DimensionTime, "time"},
		{"temperature", DimensionTemperature, "temperature"},
		{"area", DimensionArea, "area"},
		{"speed", DimensionSpeed, "speed"},
		{"energy", DimensionEnergy, "energy"},
		{"pressure", DimensionPressure, "pressure"},
		{"digital_storage", DimensionDigitalStorage, "digital_storage"},
		{"angle", DimensionAngle, "angle"},
		{"currency_per_unit", DimensionCurrencyPerUnit, "currency_per_unit"},
		{"unspecified (zero value)", DimensionUnspecified, "unspecified"},
		{"out of range", Dimension(999), "unspecified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDimension_Valid(t *testing.T) {
	if DimensionUnspecified.valid() {
		t.Error("DimensionUnspecified.valid() = true, want false")
	}
	if Dimension(999).valid() {
		t.Error("Dimension(999).valid() = true, want false")
	}
	if !DimensionMass.valid() {
		t.Error("DimensionMass.valid() = false, want true")
	}
	if !DimensionCurrencyPerUnit.valid() {
		t.Error("DimensionCurrencyPerUnit.valid() = false, want true")
	}
}

func TestNewUnit_Errors(t *testing.T) {
	z := mustDecimal("0")
	if _, err := newUnit("", DimensionMass, z, z, z, z); err == nil {
		t.Fatal("newUnit with empty symbol: expected error, got nil")
	}
	if _, err := newUnit("kg", Dimension(999), z, z, z, z); err == nil {
		t.Fatal("newUnit with invalid dimension: expected error, got nil")
	}
}

func TestMustUnit_Valid(t *testing.T) {
	z := mustDecimal("0")
	o := mustDecimal("1")
	u := mustUnit("kg", DimensionMass, o, z, o, z)
	if u.Symbol() != "kg" {
		t.Errorf("Symbol() = %q, want kg", u.Symbol())
	}
	if u.Dimension() != DimensionMass {
		t.Errorf("Dimension() = %v, want DimensionMass", u.Dimension())
	}
}

func TestMustUnit_PanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustUnit with empty symbol did not panic")
		}
	}()
	z := mustDecimal("0")
	mustUnit("", DimensionMass, z, z, z, z)
}

func TestMustDecimal_Valid(t *testing.T) {
	d := mustDecimal("49.99")
	want, err := decimal.Parse("49.99")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}
	if d.Compare(want) != 0 {
		t.Errorf("mustDecimal(%q) = %v, want %v", "49.99", d, want)
	}
}

func TestMustDecimal_PanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustDecimal(invalid) did not panic")
		}
	}()
	mustDecimal("not-a-number")
}

func TestZeroAndOne(t *testing.T) {
	if zero().Compare(mustDecimal("0")) != 0 {
		t.Errorf("zero() = %v, want 0", zero())
	}
	if one().Compare(mustDecimal("1")) != 0 {
		t.Errorf("one() = %v, want 1", one())
	}
}

func TestUnitRegistry_ZeroValue(t *testing.T) {
	var reg UnitRegistry
	if _, ok := reg.Lookup("kg"); ok {
		t.Error("Lookup on zero-value UnitRegistry: ok = true, want false")
	}
}

func TestNewUnitRegistry_LookupKnownUnits(t *testing.T) {
	reg := NewUnitRegistry()
	if reg.Version() == "" {
		t.Error("Version() = \"\", want non-empty")
	}

	cases := []struct {
		symbol string
		dim    Dimension
	}{
		{"kg", DimensionMass},
		{"g", DimensionMass},
		{"mg", DimensionMass},
		{"lb", DimensionMass},
		{"oz", DimensionMass},
		{"m", DimensionLength},
		{"km", DimensionLength},
		{"cm", DimensionLength},
		{"mm", DimensionLength},
		{"mi", DimensionLength},
		{"yd", DimensionLength},
		{"ft", DimensionLength},
		{"in", DimensionLength},
		{"L", DimensionVolume},
		{"mL", DimensionVolume},
		{"m3", DimensionVolume},
		{"gal", DimensionVolume},
		{"qt", DimensionVolume},
		{"pt", DimensionVolume},
		{"floz", DimensionVolume},
		{"s", DimensionTime},
		{"ms", DimensionTime},
		{"min", DimensionTime},
		{"h", DimensionTime},
		{"day", DimensionTime},
		{"K", DimensionTemperature},
		{"°C", DimensionTemperature},
		{"°F", DimensionTemperature},
		{"m2", DimensionArea},
		{"km2", DimensionArea},
		{"cm2", DimensionArea},
		{"ft2", DimensionArea},
		{"acre", DimensionArea},
		{"ha", DimensionArea},
		{"m/s", DimensionSpeed},
		{"km/h", DimensionSpeed},
		{"mph", DimensionSpeed},
		{"J", DimensionEnergy},
		{"kJ", DimensionEnergy},
		{"cal", DimensionEnergy},
		{"kcal", DimensionEnergy},
		{"kWh", DimensionEnergy},
		{"Pa", DimensionPressure},
		{"kPa", DimensionPressure},
		{"bar", DimensionPressure},
		{"atm", DimensionPressure},
		{"psi", DimensionPressure},
		{"B", DimensionDigitalStorage},
		{"KB", DimensionDigitalStorage},
		{"MB", DimensionDigitalStorage},
		{"GB", DimensionDigitalStorage},
		{"TB", DimensionDigitalStorage},
		{"rad", DimensionAngle},
		{"deg", DimensionAngle},
		{"grad", DimensionAngle},
		{"USD/kg", DimensionCurrencyPerUnit},
		{"USD/g", DimensionCurrencyPerUnit},
		{"USD/lb", DimensionCurrencyPerUnit},
	}
	for _, tc := range cases {
		t.Run(tc.symbol, func(t *testing.T) {
			u, ok := reg.Lookup(tc.symbol)
			if !ok {
				t.Fatalf("Lookup(%q): ok = false, want true", tc.symbol)
			}
			if u.Symbol() != tc.symbol {
				t.Errorf("Symbol() = %q, want %q", u.Symbol(), tc.symbol)
			}
			if u.Dimension() != tc.dim {
				t.Errorf("Dimension() = %v, want %v", u.Dimension(), tc.dim)
			}
		})
	}
}

func TestNewUnitRegistry_UnknownUnit(t *testing.T) {
	reg := NewUnitRegistry()
	if _, ok := reg.Lookup("flurbs"); ok {
		t.Error(`Lookup("flurbs"): ok = true, want false`)
	}
}

// approxEqual reports whether a and b differ by no more than the given
// absolute tolerance -- this file's own test assertion helper, distinct
// from (and simpler than) MU-15's relative-tolerance comparison, since
// these tests are checking this package's own conversion arithmetic
// against known physical constants, not exercising MU-15 itself.
func approxEqual(t *testing.T, a, b decimal.Decimal, tolerance string) bool {
	t.Helper()
	diff, err := a.Sub(b)
	if err != nil {
		t.Fatalf("Sub unexpected error: %v", err)
	}
	tol := mustDecimal(tolerance)
	return diff.Abs().Compare(tol) <= 0
}

func TestUnit_ToCanonical_Mass(t *testing.T) {
	reg := NewUnitRegistry()
	lb, _ := reg.Lookup("lb")
	got, err := lb.ToCanonical(mustDecimal("50"))
	if err != nil {
		t.Fatalf("ToCanonical unexpected error: %v", err)
	}
	want := mustDecimal("22.6796185") // 50 * 0.45359237, exact
	if got.Compare(want) != 0 {
		t.Errorf("50 lb -> kg = %v, want %v (exact)", got, want)
	}
}

func TestUnit_RoundTrip_Mass(t *testing.T) {
	// Round trip through the registry's own conversion factors should stay
	// well within MU-15's default 1e-9 relative tolerance for every
	// non-temperature dimension.
	reg := NewUnitRegistry()
	lb, _ := reg.Lookup("lb")
	original := mustDecimal("12")
	canonical, err := lb.ToCanonical(original)
	if err != nil {
		t.Fatalf("ToCanonical unexpected error: %v", err)
	}
	back, err := lb.FromCanonical(canonical)
	if err != nil {
		t.Fatalf("FromCanonical unexpected error: %v", err)
	}
	if !approxEqual(t, original, back, "0.000000001") {
		t.Errorf("round trip 12 lb -> kg -> lb = %v, want ~%v within 1e-9", back, original)
	}
}

func TestUnit_Temperature_Fahrenheit(t *testing.T) {
	reg := NewUnitRegistry()
	f, _ := reg.Lookup("°F")

	// 50 °F -> 283.15 K, vector 26's worked example.
	got, err := f.ToCanonical(mustDecimal("50"))
	if err != nil {
		t.Fatalf("ToCanonical unexpected error: %v", err)
	}
	want := mustDecimal("283.15")
	if !approxEqual(t, got, want, "0.00000001") {
		t.Errorf("50 °F -> K = %v, want ~%v", got, want)
	}
}

func TestUnit_Temperature_Fahrenheit_DoesNotRoundTripExactly(t *testing.T) {
	// SPEC-MU §4 MU-15, vector 97: 100 °F does not round-trip exactly,
	// because the forward conversion uses a truncated decimal
	// approximation of 5/9. This is the registry-level fact that makes
	// vector 97 possible; mu.checkMU15's own test proves the check
	// produces FAIL from it.
	reg := NewUnitRegistry()
	f, _ := reg.Lookup("°F")
	original := mustDecimal("100")
	canonical, err := f.ToCanonical(original)
	if err != nil {
		t.Fatalf("ToCanonical unexpected error: %v", err)
	}
	back, err := f.FromCanonical(canonical)
	if err != nil {
		t.Fatalf("FromCanonical unexpected error: %v", err)
	}
	if back.Compare(original) == 0 {
		t.Fatal("100 °F round-tripped exactly through °F -> K -> °F, want an inexact result (5/9 is not exactly representable as a decimal)")
	}
	// But it must still be extremely close -- within the default 1e-9
	// tolerance -- confirming the mismatch is a deliberate, tiny truncation
	// artifact, not a modelling error.
	if !approxEqual(t, original, back, "0.000000001") {
		t.Errorf("100 °F round trip = %v, drifted further than the default MU-15 tolerance from %v", back, original)
	}
}

func TestUnit_Temperature_Celsius_RoundTripsExactly(t *testing.T) {
	// Celsius's conversion is a pure offset (no irrational scale factor),
	// so unlike Fahrenheit it must round-trip exactly even at
	// tolerance "0".
	reg := NewUnitRegistry()
	c, _ := reg.Lookup("°C")
	original := mustDecimal("21")
	canonical, err := c.ToCanonical(original)
	if err != nil {
		t.Fatalf("ToCanonical unexpected error: %v", err)
	}
	back, err := c.FromCanonical(canonical)
	if err != nil {
		t.Fatalf("FromCanonical unexpected error: %v", err)
	}
	if back.Compare(original) != 0 {
		t.Errorf("21 °C round trip = %v, want exactly %v", back, original)
	}
}

func TestUnit_DimensionMismatch_NoConversionAssumed(t *testing.T) {
	// This registry never conflates two different dimensions -- "m" (length)
	// and "kg" (mass) are unrelated units. This is a sanity check that the
	// registry data itself is internally consistent (mu.checkMU04/
	// checkMU07/checkMU15 are what actually enforce this at evaluation
	// time; this test just confirms the data they read is not accidentally
	// cross-wired).
	reg := NewUnitRegistry()
	m, _ := reg.Lookup("m")
	kg, _ := reg.Lookup("kg")
	if m.Dimension() == kg.Dimension() {
		t.Fatal("m and kg report the same Dimension, want DimensionLength != DimensionMass")
	}
}

// decimalFromApd builds a decimal.Decimal near apd's own maximum exponent,
// entirely through this package's public Parse/Mul/Add surface (never
// touching apd directly), to drive ToCanonical/FromCanonical's Mul/Add
// overflow path -- the one failure mode those two methods document.
func decimalFromApd(t *testing.T) decimal.Decimal {
	t.Helper()
	// 9 followed by enough zeros to sit at apd's maximum supported
	// exponent; decimal.Parse itself has no scientific-notation escape
	// hatch, so this is built the same way decimal's own overflow tests
	// build one -- as a plain-decimal literal with apd.MaxExponent zeros.
	huge := "9" + strings.Repeat("0", int(apd.MaxExponent))
	d, err := decimal.Parse(huge)
	if err != nil {
		t.Fatalf("decimal.Parse(huge) unexpected error: %v", err)
	}
	return d
}

func TestUnit_ToCanonical_OverflowError(t *testing.T) {
	huge := decimalFromApd(t)
	u := mustUnit("huge-scale", DimensionMass, huge, zero(), one(), zero())
	if _, err := u.ToCanonical(huge); err == nil {
		t.Fatal("ToCanonical with an overflowing product: expected error, got nil")
	}
}

func TestUnit_ToCanonical_OffsetOverflowError(t *testing.T) {
	huge := decimalFromApd(t)
	// A scale of 1 keeps the multiply from overflowing on its own, so the
	// Add step -- huge + huge -- is what overflows here, isolating that
	// second failure branch from the Mul branch above.
	u := mustUnit("huge-offset", DimensionMass, one(), huge, one(), zero())
	if _, err := u.ToCanonical(huge); err == nil {
		t.Fatal("ToCanonical with an overflowing offset add: expected error, got nil")
	}
}

func TestUnit_FromCanonical_OverflowError(t *testing.T) {
	huge := decimalFromApd(t)
	u := mustUnit("huge-scale", DimensionMass, one(), zero(), huge, zero())
	if _, err := u.FromCanonical(huge); err == nil {
		t.Fatal("FromCanonical with an overflowing product: expected error, got nil")
	}
}

func TestUnit_FromCanonical_OffsetOverflowError(t *testing.T) {
	huge := decimalFromApd(t)
	u := mustUnit("huge-offset", DimensionMass, one(), zero(), one(), huge)
	if _, err := u.FromCanonical(huge); err == nil {
		t.Fatal("FromCanonical with an overflowing offset add: expected error, got nil")
	}
}
