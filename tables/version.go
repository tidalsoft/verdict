package tables

// Versioned is implemented by every reference table this package exposes.
// A caller assembling a response is expected to report which version of
// each reference table a verdict depended on (e.g. `{"iso4217": "2026-01",
// "units": "1.4.0"}`), and each table must carry a version independent of
// the others. Versioned is the single method that response-assembly code
// needs in order to build that object without a type switch over concrete
// table types: CurrencyTable and CountryTable implement it today, and a
// unit registry and an IANA tzdata table backing PG-20 are expected to
// implement it the same way, so response assembly gains a new table by
// adding a name -> Versioned entry to whatever collection it holds, not by
// changing this interface or this package.
type Versioned interface {
	// Version returns this table's version identifier. It is never empty
	// for a validly constructed table. The format is each table's own
	// convention -- a dated "YYYY-MM" for ISO 4217 and ISO 3166 here,
	// semver for a unit registry, an IANA release tag (e.g. "2026a") for
	// tzdata -- so callers must treat it as an opaque, directly reportable
	// string and never parse it.
	Version() string
}
