// Package mu implements the seven Class D magnitude/scale checks from
// SPEC-MU §3: MU-01 scale_declaration_conflict, MU-02 precision_loss,
// MU-03 currency_mismatch, MU-06 sign_convention, MU-07 range_bound,
// MU-13 percentage_domain, and MU-14 minor_unit_exponent. Each check is a
// pure function of a single field's Input; Evaluate dispatches a field to
// the checks its declared Kind carries, in the spec's internal order
// (MU-01 → MU-14 → MU-02 → MU-03 → MU-13 → MU-06 → MU-07), and returns the
// first FAIL, else the first non-PASS, else PASS.
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
// # Frozen API surface
//
// The exported surface here is the contract the check implementations
// (tasks T2-T5) build against: Input, Tables, OnFunc, and Evaluate. Each
// check is a func(Input) (verdict.Result, error) registered in the dispatch
// table keyed by field.Kind; the concrete check bodies land in later tasks,
// and this package's dispatch and aggregation logic is already real.
package mu
