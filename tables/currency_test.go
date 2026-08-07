package tables

import "testing"

func TestNewISO4217Table_Version(t *testing.T) {
	tbl := NewISO4217Table()
	if got := tbl.Version(); got != "2026-01" {
		t.Fatalf("Version() = %q, want %q", got, "2026-01")
	}
	var v Versioned = tbl
	if got := v.Version(); got != "2026-01" {
		t.Fatalf("Version() via Versioned = %q, want %q", got, "2026-01")
	}
}

func TestCurrencyTable_Lookup_ExponentClasses(t *testing.T) {
	tbl := NewISO4217Table()

	tests := []struct {
		name         string
		code         string
		wantExponent int32
	}{
		// MU-14's named examples for each exponent class.
		{"JPY exponent 0", "JPY", 0},
		{"KRW exponent 0", "KRW", 0},
		{"VND exponent 0", "VND", 0},
		{"CLP exponent 0", "CLP", 0},
		{"ISK exponent 0", "ISK", 0},
		{"USD exponent 2 (majority)", "USD", 2},
		{"EUR exponent 2 (majority)", "EUR", 2},
		{"GBP exponent 2 (majority)", "GBP", 2},
		{"CAD exponent 2 (majority)", "CAD", 2},
		{"AUD exponent 2 (majority)", "AUD", 2},
		{"BHD exponent 3", "BHD", 3},
		{"KWD exponent 3", "KWD", 3},
		{"OMR exponent 3", "OMR", 3},
		{"TND exponent 3", "TND", 3},
		{"JOD exponent 3", "JOD", 3},
		{"UYW exponent 4", "UYW", 4},
		{"CLF exponent 4", "CLF", 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := tbl.Lookup(tc.code)
			if !ok {
				t.Fatalf("Lookup(%q): not found", tc.code)
			}
			if c.Code() != tc.code {
				t.Fatalf("Code() = %q, want %q", c.Code(), tc.code)
			}
			exp, hasExp := c.MinorUnitExponent()
			if !hasExp {
				t.Fatalf("MinorUnitExponent() for %q: has = false, want true", tc.code)
			}
			if exp != tc.wantExponent {
				t.Fatalf("MinorUnitExponent() for %q = %d, want %d", tc.code, exp, tc.wantExponent)
			}
		})
	}
}

func TestCurrencyTable_Lookup_NoMinorUnit(t *testing.T) {
	tbl := NewISO4217Table()

	// XAU (gold) and XXX (no currency) are recognised ISO 4217 codes with
	// no minor unit at all -- MinorUnitExponent's ok must be false, never
	// a guessed 0 or 2.
	for _, code := range []string{"XAU", "XXX", "XDR"} {
		c, ok := tbl.Lookup(code)
		if !ok {
			t.Fatalf("Lookup(%q): not found", code)
		}
		if _, hasExp := c.MinorUnitExponent(); hasExp {
			t.Fatalf("MinorUnitExponent() for %q: has = true, want false (no minor unit in ISO 4217)", code)
		}
	}
}

func TestCurrencyTable_Lookup_Unknown(t *testing.T) {
	tbl := NewISO4217Table()

	for _, code := range []string{"ZZZ", "usd", "", "XX1"} {
		if c, ok := tbl.Lookup(code); ok {
			t.Fatalf("Lookup(%q): got %+v, ok=true; want not found", code, c)
		}
	}
}

func TestCurrencyTable_ZeroValue(t *testing.T) {
	var tbl CurrencyTable
	if _, ok := tbl.Lookup("USD"); ok {
		t.Fatalf("zero-value CurrencyTable.Lookup(%q): ok = true, want false", "USD")
	}
	if got := tbl.Version(); got != "" {
		t.Fatalf("zero-value CurrencyTable.Version() = %q, want empty", got)
	}
}
