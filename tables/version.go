package tables

// Versioned is implemented by every reference table this package exposes.
// SPEC-MU §7.2 requires each response to report a `tables` object naming
// every reference table a verdict depended on (e.g. `{"iso4217": "2026-01",
// "units": "1.4.0"}`), and §9 requires each table to carry a version
// independent of the others. Versioned is the single method
// response-assembly code (task 1-14) needs in order to build that object
// without a type switch over concrete table types: CurrencyTable and
// CountryTable implement it today, and the unit registry (task 1-5) and
// the IANA tzdata table backing PG-20 (task 1-9) are expected to implement
// it the same way, so response assembly gains a new table by adding a
// name -> Versioned entry to whatever collection it holds, not by changing
// this interface or this package.
type Versioned interface {
	// Version returns this table's version identifier. It is never empty
	// for a validly constructed table. The format is each table's own
	// convention -- a dated "YYYY-MM" for ISO 4217 and ISO 3166 here,
	// semver for a unit registry, an IANA release tag (e.g. "2026a") for
	// tzdata -- so callers must treat it as an opaque, directly reportable
	// string and never parse it.
	Version() string
}
