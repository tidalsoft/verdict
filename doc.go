// Package verdict is the Gatepost evaluation engine: the deterministic core
// that implements the magnitude/unit checks (SPEC-MU), precondition gates
// (SPEC-PG §3-5), and postcondition checks (SPEC-PG §6) that make up the
// product's evaluation surface.
//
// Purity invariant: this package and everything beneath it must perform no
// network access, no filesystem access, and no wall-clock reads. A verdict is
// a pure function of the request, the active ruleset, and any reference
// tables supplied to it; the only notion of "now" it may use is an evaluation
// timestamp passed in by the caller (SPEC-SYS §14.1). Callers needing I/O —
// HTTP handlers, storage, metering — belong in platform code that imports
// this module, never the reverse.
//
// Module boundary: this is github.com/tidalsoft/verdict, a standalone module
// published under Apache 2.0 (SPEC-SYS §14.5), extracted from the Gatepost
// monorepo's verdict/ subtree. Gatepost (proprietary, a separate repository)
// imports it; this module imports nothing from Gatepost. Because the module
// boundary itself now enforces the purity invariant mechanically (there is
// nothing under github.com/tidalsoft/gatepost/... for this module to import),
// the repository's .golangci.yml enforces it a different way: a depguard rule
// denies direct imports of packages a pure evaluation library must never
// need -- net, net/http, os, os/exec, io/ioutil, path/filepath,
// database/sql.
//
// Verdict model: this package defines the vocabulary every check and gate
// produces and that Gatepost's response contracts serialise -- the
// three-valued Outcome (SPEC-MU §2.1, SPEC-PG §2.1), check Severity and Class
// (SPEC-MU §2.2), evaluation Mode (SPEC-PG §2.2), the per-check Result, and
// the request-level Aggregate computed by ComputeAggregate. Specific checks
// (MU-*), gates (PG-*), the state envelope, and response serialisation are
// Gatepost concerns built on top of this vocabulary; they do not live here.
// The MU-, PG-, and PC- rule ID prefixes are reserved to the specification
// maintainer (SPEC-SYS §14.5); forks add rules under their own prefix.
//
// Result deliberately is not a response record, and ComputeAggregate
// deliberately does not return the per-check detail a response has to
// report. A downstream assembler must keep its own, richer per-check record
// -- pairing each Result with the Severity and Mode it was evaluated under
// -- because reconstructing a response contract's denied_under_strict_mode
// flag requires exactly the predicate ComputeAggregate applies internally
// (mode == ModeStrict && severity == SeverityBlock). Losing that pairing
// after evaluation means it cannot be reconstructed from the Aggregate
// alone.
package verdict
