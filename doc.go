// Package verdict is the Gatepost evaluation engine: the deterministic core
// that will implement the magnitude/unit checks (SPEC-MU), precondition gates
// (SPEC-PG §3-5), and postcondition checks (SPEC-PG §6) that make up the
// product's evaluation surface.
//
// Purity invariant: this package and everything beneath it must perform no
// network access, no filesystem access, and no wall-clock reads. A verdict is
// a pure function of the request, the active ruleset, and any reference
// tables supplied to it; the only notion of "now" it may use is an evaluation
// timestamp passed in by the caller (SPEC-SYS §14.1). Callers needing I/O —
// HTTP handlers, storage, metering — belong in the platform code that
// imports this package, never the reverse.
//
// Extraction plan: this package tree (github.com/tidalsoft/gatepost/verdict
// and everything beneath it) is extracted into its own standalone Apache 2.0
// repository at task 2-18, becoming its own module (github.com/tidalsoft/verdict)
// at that point. Today it
// lives in the same module as the platform code (cmd/, internal/, web/) as
// an ordinary Go subpackage — a prior separate-module design was reverted
// because it broke `go install github.com/tidalsoft/gatepost/cmd/gatepost@..`
// for end users (a module with a replace directive cannot be `go install`ed
// by anyone outside the workspace, which defeats the CLI's role as a
// discovery surface, per SPEC-LAND §5.1 / task 1-16 / 2-19).
//
// Because there is no module boundary to mechanically enforce the purity
// invariant until 2-18, it is enforced instead by a depguard rule in the
// repository's .golangci.yml: packages under verdict/... are forbidden from
// importing github.com/tidalsoft/gatepost/{cmd,internal,web}/... . That rule
// runs on every mandatory lint pass and is what keeps the 2-18 split a
// directory move rather than an import-path untangling exercise.
//
// Verdict model: this package defines the vocabulary every check and gate
// in Phase 1 produces and that the response contracts (task 1-14) will
// serialise -- the three-valued Outcome (SPEC-MU §2.1, SPEC-PG §2.1), check
// Severity and Class (SPEC-MU §2.2), evaluation Mode (SPEC-PG §2.2), the
// per-check Result, and the request-level Aggregate computed by
// ComputeAggregate. Specific checks (MU-*), gates (PG-*), the state
// envelope, and response serialisation are later tasks built on top of
// this vocabulary; they do not live here.
//
// Note for task 1-14: Result deliberately is not a response record, and
// ComputeAggregate deliberately does not return the per-check detail a
// response has to report. A downstream assembler must keep its own,
// richer per-check record -- pairing each Result with the Severity and
// Mode it was evaluated under -- because reconstructing the response
// contract's denied_under_strict_mode flag requires exactly the predicate
// ComputeAggregate applies internally (mode == ModeStrict && severity ==
// SeverityBlock). Losing that pairing after evaluation means it cannot be
// reconstructed from the Aggregate alone.
package verdict
