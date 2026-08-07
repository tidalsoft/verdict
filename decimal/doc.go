// Package decimal provides exact, arbitrary-precision decimal arithmetic for
// monetary and quantity values. It exists because money never passes
// through float64: MU-02 (precision_loss) and MU-12 (total_reconciliation)
// require monetary comparison and reconciliation to happen in exact decimal
// arithmetic, never in IEEE 754 binary64. Reconciling line items [0.1, 0.2]
// to a total of 0.3 is the tripwire this package exists to pass: that
// reconciliation is false under naive float64 addition and true under exact
// decimal addition.
//
// # Library choice
//
// This package wraps github.com/cockroachdb/apd/v3, the arbitrary-precision
// decimal library CockroachDB uses for its own SQL DECIMAL type. Every
// exported function and method here operates on this package's own Decimal
// type; no apd type appears in any exported signature. That is what keeps
// the library choice revisable: replacing apd with a different decimal
// library is a change confined to this package's unexported internals, with
// no effect on any caller.
//
// apd requires an explicit *apd.Context on every rounding-sensitive
// operation rather than encouraging a shared package-level default context
// (a pattern some decimal libraries do encourage). This package never holds
// a package-level context, precision, or rounding mode — every operation
// builds what it needs internally, so no hidden shared state can make one
// caller's rounding behavior depend on another's.
//
// # Provenance
//
// MU-02 draws a hard line between a value that arrived as a decimal string
// (exact, and the wire format required for money) and the identical numeric
// value having arrived as a JSON number (subject to binary64 fidelity
// checks: a currency amount like "0.1" or "49.99" can pass or fail
// precision_loss depending on which of these it was). This package has no
// JSON decoder and performs no I/O of any kind, so it cannot infer
// provenance from the value alone — callers state it explicitly via the
// Provenance type passed to PrecisionLoss.
package decimal
