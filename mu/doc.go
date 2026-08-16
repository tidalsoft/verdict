// Package mu implements SPEC-MU's sixteen Class D magnitude/scale/unit
// checks and three Class S statistical checks (SPEC-MU §2.2's class
// split). The Class D checks default to block severity and run
// unconditionally: MU-01 scale_declaration_conflict, MU-02
// precision_loss, MU-03 currency_mismatch, MU-06 sign_convention, MU-07
// range_bound, MU-13 percentage_domain, and MU-14 minor_unit_exponent
// (SPEC-MU §3); MU-04 unit_dimension_mismatch, MU-05 unit_absent, and
// MU-15 unit_conversion_overflow (SPEC-MU §4, though MU-15's own default
// severity is warn, not block -- see its own doc comment in
// conversion.go); and MU-08 null_vs_absent, MU-09 numeric_string_
// coercion, MU-10 timestamp_encoding, MU-11 date_range_bound, MU-12
// total_reconciliation, and MU-16 identifier_checksum (SPEC-MU §5). The
// Class S checks default to warn and reach block severity only through
// the per-field promotion path SPEC-MU §2.2 describes -- this package
// never performs that promotion itself, see statResult's doc comment in
// statistics.go: MU-20 magnitude_outlier, MU-21 scale_shift_suspect, and
// MU-22 digit_transposition_suspect (SPEC-MU §6).
//
// Each check is a pure function of a single field's Input; Evaluate
// dispatches a field to the checks its declared Kind carries, in
// ascending check-ID order (SPEC-MU §2.6), and returns every check's
// Result — evaluation does not short-circuit on a FAIL, per §2.6, so all
// applicable checks always run and all their results are always reported.
//
// SPEC-MU §2.5's "not applicable" state -- a check whose *Applies to* gate
// the field's declaration never satisfies, which contributes no entry to
// the response at all, never an INDETERMINATE one -- is distinct from
// every evaluated outcome and is implemented throughout this package; see
// OnFunc's doc comment for the exact test and Evaluate's for what a caller
// sees as a result. §2.6.3's coercion gate (a value-dependent check facing
// a value it could not read returns INDETERMINATE rather than evaluating
// against a number nobody can pin down) is implemented the same way, via
// Input.ValueCoercionFailed.
//
// # Purity invariant
//
// This package performs no network access, no filesystem access, and no
// wall-clock reads. A verdict is a pure function of the Input and the
// reference tables supplied to it. Reference tables are injected via
// Tables, never constructed in a hot path: a caller builds its ISO 4217
// table and unit registry once (e.g. alongside a ruleset) and reuses them
// across evaluations.
//
// # Result carries no evidence
//
// verdict.Result deliberately carries only checkID/class/severity/outcome —
// no evidence, no reason string, no field path. Response serialization is
// the importing service's concern, not this package's; a downstream
// assembler keeps its own richer per-check record.
//
// # API surface
//
// The exported surface here is the contract the check implementations
// build against: Input, Tables, OnFunc, and Evaluate, whose signature is
// func(Input) ([]verdict.Result, error) — one Result per applicable check,
// in dispatch order. Each individual check is a
// func(Input) (verdict.Result, bool, error) (OnFunc) registered in the
// dispatch table keyed by field.Kind, implemented in its own file alongside
// that file's tests: MU-01 (scale.go), MU-02 (precision.go), MU-03
// (currency.go), MU-04 (dimension.go), MU-05 (absent.go), MU-06 (sign.go),
// MU-07 (range.go), MU-08 (null.go), MU-09 (numeric_string.go), MU-10
// (timestamp.go), MU-11 (daterange.go), MU-12 (reconcile.go), MU-13
// (percentage.go), MU-14 (exponent.go), MU-15 (conversion.go), MU-16
// (identifier.go), MU-20 (outlier.go), MU-21 (scaleshift.go), and MU-22
// (transposition.go). resolveQuantityUnit (unit.go) is the shared unit
// resolution helper MU-04, MU-05, MU-07, and MU-15 all consult; statistics.go
// holds the decimal-exact median, quartile, and MAD arithmetic MU-20 and
// MU-21 both need, and the Class S Result constructors all three
// statistical checks build their outcomes through.
package mu
