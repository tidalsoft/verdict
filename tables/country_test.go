package tables

import "testing"

func TestNewISO3166Alpha2Table_Version(t *testing.T) {
	tbl := NewISO3166Alpha2Table()
	if got := tbl.Version(); got != "2026-08" {
		t.Fatalf("Version() = %q, want %q", got, "2026-08")
	}
	var v Versioned = tbl
	if got := v.Version(); got != "2026-08" {
		t.Fatalf("Version() via Versioned = %q, want %q", got, "2026-08")
	}
}

func TestCountryTable_Lookup_Known(t *testing.T) {
	tbl := NewISO3166Alpha2Table()

	for _, code := range []string{"US", "GB", "CA", "DE", "JP", "BR", "ZA", "CN", "AU"} {
		c, ok := tbl.Lookup(code)
		if !ok {
			t.Fatalf("Lookup(%q): not found", code)
		}
		if c.Code() != code {
			t.Fatalf("Code() = %q, want %q", c.Code(), code)
		}
	}
}

func TestCountryTable_Lookup_Unknown(t *testing.T) {
	tbl := NewISO3166Alpha2Table()

	for _, code := range []string{"ZZ", "us", "", "USA", "X1"} {
		if c, ok := tbl.Lookup(code); ok {
			t.Fatalf("Lookup(%q): got %+v, ok=true; want not found", code, c)
		}
	}
}

func TestCountryTable_ZeroValue(t *testing.T) {
	var tbl CountryTable
	if _, ok := tbl.Lookup("US"); ok {
		t.Fatalf("zero-value CountryTable.Lookup(%q): ok = true, want false", "US")
	}
	if got := tbl.Version(); got != "" {
		t.Fatalf("zero-value CountryTable.Version() = %q, want empty", got)
	}
}
