package verdict

import "fmt"

// Verdict is the request-level decision computed from every Result an
// evaluation produced (SPEC-MU §2.1, SPEC-PG §2.1-2.2).
type Verdict int

const (
	// VerdictDeny is the zero value. Deny is the safe reading for an
	// Aggregate that was never actually computed by ComputeAggregate --
	// e.g. a zero-valued struct left over from a bug -- because an unset
	// verdict must never be mistaken for permission to proceed.
	VerdictDeny Verdict = iota
	// VerdictAllowWithWarnings means at least one Result is OutcomeFail at
	// SeverityWarn, and neither of VerdictDeny's conditions holds.
	VerdictAllowWithWarnings
	// VerdictAllow means no Result is OutcomeFail, and no block-severity
	// Result is OutcomeIndeterminate under ModeStrict.
	VerdictAllow
)

// String renders the verdict using the vocabulary from SPEC-MU §2.1.
func (v Verdict) String() string {
	switch v {
	case VerdictDeny:
		return "deny"
	case VerdictAllowWithWarnings:
		return "allow_with_warnings"
	case VerdictAllow:
		return "allow"
	default:
		return "unknown"
	}
}

// DenyReason further qualifies a VerdictDeny beyond "a check failed at
// block severity," which is self-explanatory and needs no further
// qualification -- that ordinary case is the zero value.
type DenyReason int

const (
	// DenyReasonNone is the zero value: the deny, if any, needs no further
	// qualification beyond the block-severity FAIL(s) that caused it.
	DenyReasonNone DenyReason = iota
	// DenyReasonInsufficientEvidence marks a deny produced by ModeStrict
	// encountering a block-severity INDETERMINATE rather than any FAIL
	// (SPEC-PG §2.2).
	DenyReasonInsufficientEvidence
)

// String renders the deny reason using the machine-readable token from
// SPEC-PG §2.2. DenyReasonNone renders as the empty string, reflecting
// that it is not itself a reason code -- it means "no further reason
// beyond the FAIL(s) already reported."
func (d DenyReason) String() string {
	switch d {
	case DenyReasonNone:
		return ""
	case DenyReasonInsufficientEvidence:
		return "insufficient_evidence"
	default:
		return "unknown"
	}
}

// Aggregate is the computed request-level decision: the Verdict, plus a
// Reason that only carries meaning when Verdict is VerdictDeny.
type Aggregate struct {
	// Verdict is the request-level decision.
	Verdict Verdict
	// Reason further qualifies Verdict. It is DenyReasonInsufficientEvidence
	// only when ComputeAggregate produced VerdictDeny by way of ModeStrict
	// and a block-severity INDETERMINATE; it is DenyReasonNone in every
	// other case, including every VerdictAllow and
	// VerdictAllowWithWarnings, so callers must not read it as meaningful
	// unless Verdict is VerdictDeny.
	Reason DenyReason
}

// ComputeAggregate derives the request-level Aggregate from every Result an
// evaluation produced, exactly per SPEC-MU §2.1 and SPEC-PG §2.1-2.2:
//
//   - VerdictDeny if any Result is OutcomeFail at SeverityBlock, or, only
//     under ModeStrict, any Result is OutcomeIndeterminate at
//     SeverityBlock -- the latter case carries
//     DenyReasonInsufficientEvidence. A genuine block-severity FAIL takes
//     priority: if both conditions hold, Reason is DenyReasonNone. See the
//     Mode doc comment for how this treats MU- and PG-origin results
//     uniformly, a deliberate cross-spec reconciliation, not an oversight.
//   - VerdictAllowWithWarnings if any Result is OutcomeFail at
//     SeverityWarn and neither VerdictDeny condition above applies.
//   - VerdictAllow otherwise.
//
// OutcomeIndeterminate results never produce VerdictDeny under
// ModePermissive (the default), and under ModeStrict they only ever
// contribute to VerdictDeny -- never to VerdictAllow or
// VerdictAllowWithWarnings. This is the structural expression of the
// project's first invariant: INDETERMINATE never collapses to PASS.
//
// A disabled check has no Result and is simply absent from results; it has
// no effect on the Aggregate either way. An empty or nil results is
// therefore a legitimate input -- but only when it reflects zero checks
// genuinely being applicable to the request. It must never reach this
// function because ruleset resolution failed, a configured check was
// silently skipped, or an exception was swallowed upstream: those are
// errors that must be surfaced as such, before evaluation, never encoded
// as an empty []Result that this function would then read as a clean
// VerdictAllow (invariant #7: a non-2xx response is never a verdict). The
// response contract's disabled list does not cover this gap either -- it
// only records checks a customer deliberately turned off, not a
// resolution failure that produced no results at all.
//
// ComputeAggregate returns an error, and the zero Aggregate (which itself
// reads as VerdictDeny -- see the Verdict doc comment), if mode is not one
// of ModePermissive or ModeStrict, or if any Result carries an outcome or
// severity outside the values its own type defines. Both are only
// reachable by bypassing the exported constructors -- an out-of-range
// conversion, an unfilled zero-value Result, a bad config decode feeding
// Mode directly -- since NewResult and NewPromotedResult never produce
// such a value themselves. But the type system does not make that
// structurally impossible, and silently treating an unrecognised value as
// "the permissive case" would be exactly the failure mode this package
// exists to rule out. A caller that ignores the error still fails closed
// rather than open, because the zero Aggregate denies.
func ComputeAggregate(results []Result, mode Mode) (Aggregate, error) {
	if !mode.valid() {
		return Aggregate{}, fmt.Errorf("verdict: invalid mode %v", mode)
	}

	var blockFail, warnFail, blockIndeterminateStrict bool

	for _, r := range results {
		if !r.outcome.valid() {
			return Aggregate{}, fmt.Errorf("verdict: check %q: invalid outcome %v", r.checkID, r.outcome)
		}
		if !r.severity.valid() {
			return Aggregate{}, fmt.Errorf("verdict: check %q: invalid severity %v", r.checkID, r.severity)
		}

		switch r.outcome {
		case OutcomeFail:
			if r.severity == SeverityBlock {
				blockFail = true
			} else {
				warnFail = true
			}
		case OutcomeIndeterminate:
			if mode == ModeStrict && r.severity == SeverityBlock {
				blockIndeterminateStrict = true
			}
		case OutcomePass:
			// No effect on the aggregate.
		}
	}

	if blockFail {
		return Aggregate{Verdict: VerdictDeny}, nil
	}
	if blockIndeterminateStrict {
		return Aggregate{Verdict: VerdictDeny, Reason: DenyReasonInsufficientEvidence}, nil
	}
	if warnFail {
		return Aggregate{Verdict: VerdictAllowWithWarnings}, nil
	}
	return Aggregate{Verdict: VerdictAllow}, nil
}
