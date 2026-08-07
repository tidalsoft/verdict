// Package engine is the Gatepost evaluation engine: the deterministic core
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
// Extraction plan: this package tree (github.com/evanisnor/gatepost/engine
// and everything beneath it) is extracted into its own standalone Apache 2.0
// repository at task 2-18, becoming its own module at that point. Today it
// lives in the same module as the platform code (cmd/, internal/, web/) as
// an ordinary Go subpackage — a prior separate-module design was reverted
// because it broke `go install github.com/evanisnor/gatepost/cmd/gatepost@..`
// for end users (a module with a replace directive cannot be `go install`ed
// by anyone outside the workspace, which defeats the CLI's role as a
// discovery surface, per SPEC-LAND §5.1 / task 1-16 / 2-19).
//
// Because there is no module boundary to mechanically enforce the purity
// invariant until 2-18, it is enforced instead by a depguard rule in the
// repository's .golangci.yml: packages under engine/... are forbidden from
// importing github.com/evanisnor/gatepost/{cmd,internal,web}/... . That rule
// runs on every mandatory lint pass and is what keeps the 2-18 split a
// directory move rather than an import-path untangling exercise.
//
// This package is scaffolding: it currently exposes only build metadata.
// Verdict types, checks, and gates land as the engine is implemented.
package engine
