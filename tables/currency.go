package tables

// Currency is a single ISO 4217 entry (SPEC-MU MU-14 minor_unit_exponent,
// MU-16 identifier_checksum's iso4217 scheme). Its zero value is not a
// meaningful currency; values are only ever produced by CurrencyTable.
// Lookup.
type Currency struct {
	code                 string
	minorUnitExponent    int32
	hasMinorUnitExponent bool
}

// Code returns the currency's three-letter ISO 4217 alphabetic code (e.g.
// "USD").
func (c Currency) Code() string { return c.code }

// MinorUnitExponent returns how many digits follow the decimal point in
// this currency's minor unit -- 2 for USD, 0 for JPY, 3 for KWD, 4 for
// UYW/CLF -- and whether it has one at all.
//
// The second return value is false for a handful of ISO 4217 entries that
// are not everyday transactable currencies: precious metals (XAU, XAG,
// XPD, XPT), bond-market funds codes (XBA-XBD, XSU, XUA), the IMF's
// Special Drawing Right (XDR), the reserved test code (XTS), and the "no
// currency involved" sentinel (XXX). ISO 4217 itself declares no minor
// unit for these ("N.A." in the published list), and MU-14 must not
// mistake that absence for exponent 0 -- 0 is itself a legitimate, common
// exponent (JPY, KRW, VND, CLP, ISK all carry it).
func (c Currency) MinorUnitExponent() (int32, bool) {
	return c.minorUnitExponent, c.hasMinorUnitExponent
}

// CurrencyTable is an immutable, versioned ISO 4217 currency lookup table
// (SPEC-MU §9). Its zero value is not usable -- construct one with
// NewISO4217Table. A CurrencyTable is safe for concurrent use, since
// nothing about it is ever mutated after NewISO4217Table returns.
type CurrencyTable struct {
	version string
	byCode  map[string]Currency
}

// Version implements Versioned, reporting ISO 4217 Table A.1's own
// publication date (see verdict/tables/generate/iso4217).
func (t CurrencyTable) Version() string { return t.version }

// Lookup returns the Currency for code and whether code is present in the
// table at all. Matching is exact: ISO 4217 codes are uppercase by
// convention, and Lookup does not case-fold, so a caller whose input
// source might supply lowercase or mixed-case codes must normalise before
// calling.
//
// A false second return value is what MU-16's iso4217 scheme and MU-03/
// MU-14's currency resolution must treat as "unrecognised": SPEC-MU MU-16
// requires an unrecognised or unresolvable currency to evaluate as
// INDETERMINATE, never PASS, and never a guessed exponent.
func (t CurrencyTable) Lookup(code string) (Currency, bool) {
	c, ok := t.byCode[code]
	return c, ok
}

// NewISO4217Table builds the compiled-in ISO 4217 currency table. It is a
// pure function of this package's compiled-in data (currency_data.go,
// generated -- see verdict/tables/generate/iso4217) and allocates a lookup
// map on every call, so callers should build one table once (e.g.
// alongside a ruleset or evaluator) and reuse it across evaluations rather
// than calling this per request -- see the package doc comment.
func NewISO4217Table() CurrencyTable {
	entries := iso4217Rows()
	byCode := make(map[string]Currency, len(entries))
	for _, c := range entries {
		byCode[c.code] = c
	}
	return CurrencyTable{version: iso4217Version, byCode: byCode}
}
