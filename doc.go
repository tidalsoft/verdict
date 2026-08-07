// Package verdict is a deterministic evaluation engine: it implements
// magnitude/unit checks, precondition gates, and postcondition checks over
// a request's declared schema.
//
// Purity invariant: this package and everything beneath it must perform no
// network access, no filesystem access, and no wall-clock reads. A verdict is
// a pure function of the request, the active ruleset, and any reference
// tables supplied to it; the only notion of "now" it may use is an evaluation
// timestamp passed in by the caller. Callers needing I/O — HTTP handlers,
// storage, metering — belong in platform code that imports this module,
// never the reverse.
//
// Module boundary: this is github.com/tidalsoft/verdict, a standalone module
// published under Apache 2.0. The purity invariant is enforced mechanically
// by the module boundary itself, and the repository's .golangci.yml enforces
// it a different way: a depguard rule denies direct imports of packages a
// pure evaluation library must never need -- net, net/http, os, os/exec,
// io/ioutil, path/filepath, database/sql.
//
// Verdict model: this package defines the vocabulary every check and gate
// produces -- the three-valued Outcome, check Severity and Class, evaluation
// Mode, the per-check Result, and the request-level Aggregate computed by
// ComputeAggregate. Specific checks (MU-*), gates (PG-*), and anything that
// serialises a response are concerns of the code that imports this package;
// they do not live here. The MU-, PG-, and PC- rule ID prefixes are reserved
// to the specification maintainer; forks add rules under their own prefix.
//
// Result deliberately is not a response record, and ComputeAggregate
// deliberately does not return the per-check detail a response has to
// report. A downstream assembler must keep its own, richer per-check record
// -- pairing each Result with the Severity and Mode it was evaluated under
// -- because reconstructing a response's "denied under strict mode" flag
// requires exactly the predicate ComputeAggregate applies internally
// (mode == ModeStrict && severity == SeverityBlock). Losing that pairing
// after evaluation means it cannot be reconstructed from the Aggregate
// alone.
package verdict
