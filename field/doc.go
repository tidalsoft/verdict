// Package field defines the typed per-field declaration a ruleset supplies
// for a single argument path (SPEC-MU §2.3) -- the schema Class D checks
// (MU-*, tasks 1-4..1-7) evaluate an argument against, as opposed to the
// versioned reference data (tables) those checks also consult.
//
// # Absence is not a default
//
// SPEC-MU §2.3 is explicit: "Where a declaration is absent, Class D checks
// depending on it return INDETERMINATE. They do not guess." This package
// makes that distinction structural at two levels, both via Go's
// comma-ok idiom rather than a sentinel zero value:
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
// SPEC-MU §2.3 names six kinds -- money, quantity, timestamp, percentage,
// decimal, identifier -- each with its own attribute vocabulary
// (currency_field and scale belong to money; dimension and unit_field to
// quantity; and so on). Modelling this as one struct with every attribute
// as an optional field would let a ruleset declare `dimension` on a
// `kind: money` field, a combination no check will ever look at and that
// SPEC-MU never describes. Instead, Declaration is an interface
// implemented by six concrete types (MoneyDeclaration,
// QuantityDeclaration, TimestampDeclaration, PercentageDeclaration,
// DecimalDeclaration, IdentifierDeclaration), one per Kind; a caller
// recovers the concrete type via a type switch (or Kind() first, to choose
// the branch) to reach kind-specific accessors. The one attribute SPEC-MU
// does not scope to a single kind -- null_semantics (MU-08) -- is
// therefore exposed on the Declaration interface itself, implemented
// identically by all six concrete types via the embedded, unexported
// common type.
//
// # What this package deliberately does not validate
//
// A Declaration's constructors and With* methods validate the shape of a
// single attribute (e.g. WithScale rejects an out-of-range Scale), but
// never cross-attribute or cross-field relationships: WithMin and WithMax
// do not check min <= max, and WithSign/WithSignWhen do not reject a
// Declaration that (unusually) sets both. Those are ruleset-authoring
// concerns (SPEC-SYS §8.2 validation rejections, task 1-12), not schema
// shape -- this package's job is to represent faithfully what a ruleset
// declared, not to judge whether the declaration makes evaluative sense.
// Likewise, Dimension (quantity) and Scheme (identifier) are stored as
// plain strings rather than closed enums: resolving a dimension requires
// the unit registry (task 1-5), and resolving a scheme requires MU-16 to
// be able to report an *unrecognised* value as INDETERMINATE rather than
// have this package reject it before MU-16 ever runs.
package field
