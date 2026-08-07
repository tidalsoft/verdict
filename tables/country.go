package tables

// Country is a single ISO 3166-1 alpha-2 entry (SPEC-MU MU-16
// identifier_checksum's iso3166_alpha2 scheme). Its zero value is not a
// meaningful country; values are only ever produced by CountryTable.
// Lookup.
type Country struct {
	code string
}

// Code returns the country's two-letter ISO 3166-1 alpha-2 code (e.g.
// "CA").
func (c Country) Code() string { return c.code }

// CountryTable is an immutable, versioned ISO 3166-1 alpha-2 lookup table
// (SPEC-MU §9). Its zero value is not usable -- construct one with
// NewISO3166Alpha2Table. A CountryTable is safe for concurrent use, since
// nothing about it is ever mutated after NewISO3166Alpha2Table returns.
type CountryTable struct {
	version string
	byCode  map[string]Country
}

// Version implements Versioned -- see engine/tables/generate/iso3166 for
// what this date represents (a cross-verification date, not a publication
// date: ISO 3166-1 alpha-2 has no free primary-source machine-readable
// release to cite the way ISO 4217 does).
func (t CountryTable) Version() string { return t.version }

// Lookup returns the Country for code and whether code is present in the
// table at all. Matching is exact: ISO 3166-1 alpha-2 codes are uppercase
// by convention, and Lookup does not case-fold, so a caller whose input
// source might supply lowercase or mixed-case codes must normalise before
// calling.
//
// A false second return value is what MU-16's iso3166_alpha2 scheme must
// treat as "unrecognised": SPEC-MU MU-16 requires an unrecognised or
// unresolvable code to evaluate as INDETERMINATE, never PASS.
func (t CountryTable) Lookup(code string) (Country, bool) {
	c, ok := t.byCode[code]
	return c, ok
}

// NewISO3166Alpha2Table builds the compiled-in ISO 3166-1 alpha-2 country
// table. It is a pure function of this package's compiled-in data
// (country_data.go, generated -- see engine/tables/generate/iso3166) and
// allocates a lookup map on every call, so callers should build one table
// once and reuse it across evaluations rather than calling this per
// request -- see the package doc comment.
func NewISO3166Alpha2Table() CountryTable {
	codes := iso3166Alpha2Rows()
	byCode := make(map[string]Country, len(codes))
	for _, code := range codes {
		byCode[code] = Country{code: code}
	}
	return CountryTable{version: iso3166Version, byCode: byCode}
}
