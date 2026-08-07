// Package decimal provides exact, arbitrary-precision decimal arithmetic for
// the Gatepost engine. It exists because money never floats (CLAUDE.md
// project-specific invariant #2): SPEC-MU MU-02 (precision_loss) and MU-12
// (total_reconciliation) require monetary comparison and reconciliation to
// happen in exact decimal arithmetic, never in IEEE 754 binary64. SPEC-MU §8
// vector 33 — line items [0.1, 0.2] reconciling to a total of 0.3 — is the
// tripwire this package exists to pass: that reconciliation is false under
// naive float64 addition and true under exact decimal addition, and SPEC-MU
// calls it "the single most important test in this document."
//
// # Library choice
//
// This package wraps github.com/cockroachdb/apd/v3, the arbitrary-precision
// decimal library CockroachDB uses for its own SQL DECIMAL type. Every
// exported function and method here operates on this package's own Decimal
// type; no apd type appears in any exported signature. That is what keeps
// the library choice revisable (PLAN.md task 1-3): replacing apd with a
// different decimal library is a change confined to this package's
// unexported internals, with no effect on any caller.
//
// apd requires an explicit *apd.Context on every rounding-sensitive
// operation rather than encouraging a shared package-level default context
// (a pattern some decimal libraries do encourage). This package never holds
// a package-level context, precision, or rounding mode — every operation
// builds what it needs internally — in keeping with this codebase's "no
// package-level state" rule (CLAUDE.md, Engineering principles).
//
// # Provenance
//
// SPEC-MU MU-02 draws a hard line between a value that arrived as a decimal
// string (exact, and the wire format SPEC-SYS §5.1 requires for money) and
// the identical numeric value having arrived as a JSON number (subject to
// binary64 fidelity checks: SPEC-MU §8 vectors 9 and 10 require opposite
// verdicts for the same value "0.1"/"49.99" depending on which of these it
// was). This package has no JSON decoder and performs no I/O of any kind, so
// it cannot infer provenance from the value alone — callers state it
// explicitly via the Provenance type passed to PrecisionLoss.
package decimal
