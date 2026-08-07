package verdict

import (
	"errors"
	"fmt"
)

// Result is the outcome of evaluating a single check or gate: which check
// ran (CheckID), what class it belongs to, at what severity a FAIL (or, in
// ModeStrict, an INDETERMINATE) counts, and what the check found (Outcome).
// It is the unit of input ComputeAggregate consumes.
//
// A Result is only ever constructed for a check that actually evaluated.
// There is deliberately no representation of "disabled" here: a disabled
// check contributes no Result at all, so code assembling the []Result
// passed to ComputeAggregate simply omits it (see the Severity doc
// comment).
//
// The zero Result is not a usable value -- its checkID is empty, and both
// NewResult and NewPromotedResult refuse to construct a Result with an
// empty checkID. Result values are only ever produced by those two
// constructors, so a valid Result is always fully populated.
type Result struct {
	checkID  string
	class    Class
	severity Severity
	outcome  Outcome
}

// CheckID returns the identifier of the check or gate that produced this
// result (e.g. "MU-01", "PG-05").
func (r Result) CheckID() string { return r.checkID }

// Class returns the check class (SPEC-MU §2.2) that produced this result.
func (r Result) Class() Class { return r.class }

// Severity returns the severity this result's FAIL (or, in ModeStrict,
// INDETERMINATE) counts at.
func (r Result) Severity() Severity { return r.severity }

// Outcome returns what the check found.
func (r Result) Outcome() Outcome { return r.outcome }

// NewResult constructs a Result for a check or gate that evaluated at the
// given class and severity. checkID must be non-empty; class must be
// ClassD or ClassS; severity must be SeverityWarn or SeverityBlock;
// outcome must be one of the three defined Outcome values.
//
// A Class S check cannot be constructed at block severity through this
// constructor: Class S defaults to warn (SPEC-MU §2.2) and reaches block
// only for a specific field, only after a measured per-field promotion.
// This package has no visibility into fire counts or precision
// measurements -- that lifecycle is task 2-14, platform-side -- and so
// cannot itself verify a promotion is warranted; what it can do is refuse
// the shortcut of reaching block by accident through the ordinary
// constructor. Use NewPromotedResult for a check whose promotion has
// already been authorized upstream.
func NewResult(checkID string, class Class, severity Severity, outcome Outcome) (Result, error) {
	if checkID == "" {
		return Result{}, errors.New("verdict: check id must not be empty")
	}
	if !class.valid() {
		return Result{}, fmt.Errorf("verdict: check %q: invalid check class %v", checkID, class)
	}
	if !severity.valid() {
		return Result{}, fmt.Errorf("verdict: check %q: invalid severity %v", checkID, severity)
	}
	if !outcome.valid() {
		return Result{}, fmt.Errorf("verdict: check %q: invalid outcome %v", checkID, outcome)
	}
	if class == ClassS && severity == SeverityBlock {
		return Result{}, fmt.Errorf("verdict: check %q: class S check cannot be constructed at block severity via NewResult; use NewPromotedResult once promotion has been authorized upstream (SPEC-MU §2.2)", checkID)
	}
	return Result{checkID: checkID, class: class, severity: severity, outcome: outcome}, nil
}

// NewPromotedResult constructs a Result for a Class S check whose severity
// has already been promoted to block by an authorized, measured,
// platform-side decision (SPEC-MU §2.2; the promotion lifecycle itself is
// task 2-14, out of scope here). This package cannot verify that such a
// decision was made correctly -- it performs no I/O and sees only what the
// caller passes it -- so calling this constructor is itself the caller's
// assertion that promotion has already happened. Expected callers are
// resolved-ruleset evaluation code acting on a ruleset that already
// reflects the promotion, never check implementations deciding severity
// for themselves.
func NewPromotedResult(checkID string, outcome Outcome) (Result, error) {
	if checkID == "" {
		return Result{}, errors.New("verdict: check id must not be empty")
	}
	if !outcome.valid() {
		return Result{}, fmt.Errorf("verdict: check %q: invalid outcome %v", checkID, outcome)
	}
	return Result{checkID: checkID, class: ClassS, severity: SeverityBlock, outcome: outcome}, nil
}
