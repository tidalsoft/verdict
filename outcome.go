package engine

// Outcome is the three-valued result of evaluating a single check or gate
// (SPEC-MU §2.1, SPEC-PG §2.1).
//
// The zero value is OutcomeIndeterminate, not OutcomePass. This is
// deliberate: a zero-initialized Outcome -- a struct field left unset by a
// bug, a map lookup miss, a variable declared but not yet assigned -- must
// read as "not evaluated," which is the safe interpretation, never as
// "evaluated and found nothing wrong." Reporting success while having
// verified nothing is the specific silent-failure mode that both SPEC-MU
// §2.1 and SPEC-PG §2.1 (which calls the equivalent rule "the single most
// important requirement in this document") exist to prevent.
type Outcome int

const (
	// OutcomeIndeterminate is the zero value: the check could not evaluate
	// because a required declaration, schema, or piece of state was
	// absent. It must never be treated as a pass -- see the Outcome doc
	// comment -- and it never changes an Aggregate's Verdict in
	// ModePermissive (the default).
	OutcomeIndeterminate Outcome = iota
	// OutcomePass means the check evaluated and found no violation.
	OutcomePass
	// OutcomeFail means the check evaluated and found a violation.
	OutcomeFail
)

// String renders the outcome using the vocabulary from SPEC-MU §2.1 /
// SPEC-PG §2.1. An out-of-range value (only reachable via an explicit
// conversion such as Outcome(7)) renders as "UNKNOWN_OUTCOME" rather than
// panicking, since fmt.Stringer implementations must not panic.
func (o Outcome) String() string {
	switch o {
	case OutcomeIndeterminate:
		return "INDETERMINATE"
	case OutcomePass:
		return "PASS"
	case OutcomeFail:
		return "FAIL"
	default:
		return "UNKNOWN_OUTCOME"
	}
}

// valid reports whether o is one of the three defined Outcome values.
// Unexported: it exists only to let NewResult and NewPromotedResult refuse
// an out-of-range value (e.g. Outcome(7)) that Go's type system does not
// itself prevent.
func (o Outcome) valid() bool {
	return o == OutcomeIndeterminate || o == OutcomePass || o == OutcomeFail
}
