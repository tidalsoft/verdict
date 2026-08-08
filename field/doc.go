// Package field defines the typed per-field declaration a ruleset supplies
// for a single argument path -- the schema Class D checks (MU-*) evaluate
// an argument against, as opposed to the versioned reference data (tables)
// those checks also consult.
//
// # Absence is not a default
//
// The design principle here is explicit: where a declaration is absent,
// Class D checks depending on it return INDETERMINATE. They do not guess.
// This package makes that distinction structural at two levels, both via
// Go's comma-ok idiom rather than a sentinel zero value:
//
//   - Field-level: Registry.Lookup's second return value is false when a
//     field path was never declared at all. There is no way to construct a
//     Registry entry that looks like "no declaration" -- the only way a
//     path is absent from a Registry is that it was never passed to
//     NewRegistry.
//   - Attribute-level: within a Declaration that does exist, most
//     accessors (e.g. MoneyDeclaration.Scale) return (value, bool) rather
//     than a bare value. A ruleset can legitimately declare `kind: money`
//     without a `scale`, and MU-01 must tell that apart from a scale that
//     happens to equal whatever Go's zero value for Scale would render as.
//     Every such accessor's zero-value ok is false, never true -- see each
//     concrete type's field comments.
//
// # A closed sum type, not one struct with optional fields
//
// This package defines six kinds -- money, quantity, timestamp, percentage,
// decimal, identifier -- each with its own attribute vocabulary
// (currency_field and scale belong to money; dimension and unit_field to
// quantity; and so on). Modelling this as one struct with every attribute
// as an optional field would let a ruleset declare `dimension` on a
// `kind: money` field, a combination no check will ever look at and that
// has no defined meaning. Instead, Declaration is an interface implemented
// by six concrete types (MoneyDeclaration, QuantityDeclaration,
// TimestampDeclaration, PercentageDeclaration, DecimalDeclaration,
// IdentifierDeclaration), one per Kind; a caller recovers the concrete type
// via a type switch (or Kind() first, to choose the branch) to reach
// kind-specific accessors. The one attribute that isn't scoped to a single
// kind -- null_semantics (MU-08) -- is therefore exposed on the Declaration
// interface itself, implemented identically by all six concrete types via
// the embedded, unexported common type.
//
// # What this package deliberately does not validate
//
// A Declaration's constructors and With* methods validate the shape of a
// single attribute (e.g. WithScale rejects an out-of-range Scale), but
// never cross-attribute or cross-field relationships: WithMin and WithMax
// do not check min <= max, and WithSign/WithSignWhen do not reject a
// Declaration that (unusually) sets both. Those are ruleset-authoring
// concerns, not schema shape -- this package's job is to represent
// faithfully what a ruleset declared, not to judge whether the declaration
// makes evaluative sense. Likewise, Dimension (quantity) and Scheme
// (identifier) are stored as plain strings rather than closed enums:
// resolving a dimension requires a unit registry this library does not
// provide, and resolving a scheme requires MU-16 to be able to report an
// *unrecognised* value as INDETERMINATE rather than have this package
// reject it before MU-16 ever runs.
//
// # Adding a new Kind
//
// The six kinds above are a closed set, but the set is not frozen: a new
// Kind is a deliberate, five-step addition, and the steps below are the
// whole of it. Follow them in order; each step has a check that fails
// loudly if it was skipped.
//
//  1. Add the KindX constant to the Kind enum in kind.go. The zero value
//     must stay KindUnspecified: insert the new constant after
//     KindIdentifier, never before KindUnspecified, so a zero-initialized
//     Kind keeps reading as "not set" rather than silently aliasing the
//     new kind.
//  2. Add the corresponding case to Kind.String(). The name returned
//     there is the name a ruleset writes as `kind: x`, so it must match
//     the new kind's canonical name exactly.
//  3. Create x.go with the concrete XDeclaration type implementing
//     Declaration. Mirror money.go's shape exactly: embed the unexported
//     common type (which supplies null_semantics, MU-08), give every
//     attribute a comma-ok accessor returning (T, bool) whose ok is false
//     until the corresponding With* method is called, and give every
//     With* method the fluent signature (XDeclaration, error) that
//     returns the zero XDeclaration on error, never a partially-mutated
//     value. The constructor is NewXDeclaration returning XDeclaration
//     with no attributes declared; Kind() returns the new KindX.
//  4. Create x_test.go with >=100% coverage of the new type. This is the
//     repo's gate: `make check` enforces 100% file, package, and total
//     coverage with zero exclusions, so a new type without a colocated
//     test fails the gate. Cover every accessor's absent and present
//     paths, every With* error path, and the Kind()/String() round trip.
//  5. The Registry needs no change. It is generic over Declaration
//     (map[string]Declaration), so a new concrete type flows through
//     NewRegistry and Lookup without a single edit.
//
// # Codegen is deferred, not adopted
//
// The five steps above are mechanical, and a reader may wonder why they
// are not generated. They are not, and that is a decision, not an
// oversight: codegen is NOT adopted today. The six existing types are
// written, reviewed, and at 100% coverage; a generator would have to
// reproduce that exact shape to earn its keep, and the cost of a seventh
// hand-written type is one file and one test.
//
// The decision is deferred until a 7th Kind appears. At that point, build
// a generator following the tables/generate precedent: a main.go guarded
// by `//go:build ignore` in a generate/ subdirectory, run via `go run`,
// with its source data committed alongside the generator, and regenerate
// all six existing types plus the new one so the output stays uniform.
// Note the mechanism precisely: this repo has no `go generate` mechanism
// anywhere, and the precedent is `//go:build ignore` + `go run`, not the
// `go generate` command. A generator must follow the precedent that exists.
package field
