package verdict

// Mode selects how ComputeAggregate treats an INDETERMINATE outcome from a
// block-severity check.
//
// Design note: ComputeAggregate applies Mode uniformly to every
// block-severity INDETERMINATE, regardless of which family of check
// produced it, so that evaluation mode behaves as a single per-decision
// setting rather than varying result by result. This is a deliberate
// choice, not an oversight, made so a caller only has to reason about one
// mode value per evaluation.
type Mode int

const (
	// ModePermissive is the zero value and the documented default:
	// INDETERMINATE never affects the Verdict. Making the
	// default itself the zero value is safe here -- unlike Outcome, where
	// a zero value that looked like success would hide a blind spot,
	// permissive mode does not hide INDETERMINATE results, it only
	// declines to let them deny the request by themselves. Reporting them
	// separately is the responsibility of whatever calls ComputeAggregate
	// and inspects the underlying []Result, not of this type.
	ModePermissive Mode = iota
	// ModeStrict denies the request when any block-severity check returns
	// INDETERMINATE, with Aggregate.Reason set to
	// DenyReasonInsufficientEvidence. It is the correct
	// setting for financial and irreversible operations, and is not the
	// default because a product that denies heavily on day one -- before
	// a customer's state supply is complete -- gets uninstalled on day
	// two.
	ModeStrict
)

// String renders the mode's canonical name.
func (m Mode) String() string {
	switch m {
	case ModePermissive:
		return "permissive"
	case ModeStrict:
		return "strict"
	default:
		return "unknown"
	}
}

// valid reports whether m is one of the two legitimate modes. Unexported:
// it exists so ComputeAggregate can refuse an out-of-range Mode -- a bad
// config decode, a future third mode added without updating this package --
// with an error instead of silently treating it as ModePermissive, which
// would be the same silent-degrade-to-permissive failure this package
// exists to rule out at the Outcome level.
func (m Mode) valid() bool {
	return m == ModePermissive || m == ModeStrict
}
