package verdict

// Severity classifies how a FAIL outcome (or, in ModeStrict, an
// INDETERMINATE outcome) at a given check affects the aggregate verdict
// (SPEC-MU §2.1, SPEC-PG §2.2).
//
// Severity applies only to a Result that was actually constructed for a
// check that evaluated. A disabled check is neither block nor warn: it
// runs at no severity at all, because it produces no Result in the first
// place. "Disabled" is deliberately not a third value of this type --
// keeping it that way means ComputeAggregate never has to reason about a
// disabled sentinel leaking in among real block/warn values; callers
// assembling the []Result to aggregate simply omit disabled checks
// entirely.
type Severity int

const (
	// SeverityUnspecified is the zero value. It is not a legitimate
	// severity for an evaluated check -- NewResult and NewPromotedResult
	// both reject it -- and exists so that a zero-initialized Severity
	// reads as "not set" rather than silently aliasing warn or block.
	SeverityUnspecified Severity = iota
	// SeverityWarn means a FAIL contributes to VerdictAllowWithWarnings
	// (absent any block-severity FAIL) but never denies by itself.
	SeverityWarn
	// SeverityBlock means a FAIL denies the request outright. In
	// ModeStrict, an INDETERMINATE at this severity also denies
	// (SPEC-PG §2.2).
	SeverityBlock
)

// String renders the severity using the vocabulary from SPEC-MU §2.1.
func (s Severity) String() string {
	switch s {
	case SeverityWarn:
		return "warn"
	case SeverityBlock:
		return "block"
	default:
		return "unspecified"
	}
}

// valid reports whether s is one of the two legitimate severities. The
// zero value, SeverityUnspecified, is not valid -- see the Severity doc
// comment. Unexported: it exists so NewResult, NewPromotedResult, and
// ComputeAggregate can refuse an out-of-range or zero-value Severity
// instead of silently treating it as one of the real values.
func (s Severity) valid() bool {
	return s == SeverityWarn || s == SeverityBlock
}
