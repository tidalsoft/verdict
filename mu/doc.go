// Package mu implements the seven Class D magnitude/scale checks from
// SPEC-MU §3: MU-01 scale_declaration_conflict, MU-02 precision_loss,
// MU-03 currency_mismatch, MU-06 sign_convention, MU-07 range_bound,
// MU-13 percentage_domain, and MU-14 minor_unit_exponent. Each check is a
// pure function of a single field's Input; Evaluate dispatches a field to
// the checks its declared Kind carries, in ascending check-ID order
// (SPEC-MU §2.4), and returns every check's Result — evaluation does not
// short-circuit on a FAIL, per §2.4, so all applicable checks always run
// and all their results are always reported.
//
// # Purity invariant
//
// This package performs no network access, no filesystem access, and no
// wall-clock reads. A verdict is a pure function of the Input and the
// reference tables supplied to it. Reference tables are injected via
// Tables, never constructed in a hot path: a caller builds its ISO 4217
// table once (e.g. alongside a ruleset) and reuses it across evaluations.
//
// # Result carries no evidence
//
// verdict.Result deliberately carries only checkID/class/severity/outcome —
// no evidence, no reason string, no field path. Response serialization is
// the importer's concern (gatepost), not this package's; a downstream
// assembler keeps its own richer per-check record.
//
// # API surface
//
// The exported surface here is the contract the check implementations
// build against: Input, Tables, OnFunc, and Evaluate, whose signature is
// func(Input) ([]verdict.Result, error) — one Result per applicable check,
// in dispatch order. Each individual check is a
// func(Input) (verdict.Result, error) registered in the dispatch table
// keyed by field.Kind, implemented in its own file alongside that file's
// tests: MU-01 (scale.go), MU-02 (precision.go), MU-03 (currency.go),
// MU-06 (sign.go), MU-07 (range.go), MU-13 (percentage.go), and MU-14
// (exponent.go).
package mu
