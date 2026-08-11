// Package tables provides versioned reference tables: compiled-in,
// immutable lookup data that Class D checks consult instead of guessing.
// ISO 4217 currency minor-unit exponents (MU-14), ISO 3166-1 alpha-2
// country codes (MU-16), and a unit registry (MU-04/MU-05/MU-07/MU-15)
// live here today; an IANA tzdata table backing PG-20 is expected to join
// this package's pattern without restructuring it -- see the Versioned
// interface.
//
// # Purity and data sourcing
//
// The engine performs no network or filesystem access at evaluation time,
// so every table here is compiled into the binary as ordinary Go source
// rather than loaded from a file or fetched at runtime. For CurrencyTable
// and CountryTable that source is generated, not hand-transcribed, from a
// committed snapshot of an external authority -- see tables/generate/iso4217
// and tables/generate/iso3166 for exactly what was fetched, when, and how
// each generator turns it into the *_data.go file it produces. Regenerating
// one of those two tables means re-running the relevant generator against a
// freshly fetched source, never hand-editing its *_data.go file.
//
// UnitRegistry (unit_data.go) is the deliberate exception: there is no
// single external authority a generator could snapshot the way ISO 4217
// and ISO 3166 are single authorities, so its data is hand-curated
// directly in source, versioned like any other change under this module's
// own release discipline rather than regenerated. unit_data.go's own doc
// comment states this and explains where each constant came from; nothing
// about that file's editing process resembles currency_data.go's or
// country_data.go's.
//
// # No package-level state
//
// Every table type (CurrencyTable, CountryTable) is built by a
// constructor (NewISO4217Table, NewISO3166Alpha2Table) that returns an
// ordinary value holding an unexported map. There is deliberately no
// package-level `var` holding that map: a package-level map is mutable by
// any importer that can reach it (Go gives no way to expose a read-only
// view of map internals without wrapping it), which is exactly the kind of
// shared, externally-mutable state this package avoids by construction.
// Building the map inside a constructor instead keeps the mutable step --
// map construction -- entirely inside this package, and the returned
// CurrencyTable/CountryTable value exposes no method that can mutate it
// back out: Lookup reads, nothing writes. A caller is expected to build a
// table once (typically alongside the ruleset or evaluator that will use
// it) and reuse the returned value across evaluations, since a low-latency
// evaluation budget has no room for rebuilding a ~180-or-249-entry map on
// every request.
package tables
