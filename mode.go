package verdict

// Mode selects how ComputeAggregate treats an INDETERMINATE outcome from a
// block-severity check (SPEC-PG §2.2).
//
// Cross-spec note (recorded per CLAUDE.md rather than resolved silently):
// SPEC-MU §2.1 states unconditionally, with no mode qualifier, that
// INDETERMINATE results never change the aggregate verdict; strict mode is
// defined only in SPEC-PG §2.2, scoped there to "a gate whose severity is
// block." ComputeAggregate applies Mode uniformly to every block-severity
// INDETERMINATE regardless of whether it came from an MU check or a PG
// gate, on the basis that SPEC-SYS models evaluation mode as a single
// per-decision field and emits one evaluation-wide
// check.denied_insufficient_evidence event covering both documents. Read
// against SPEC-MU alone, this reads as a deviation; it is deliberate, not
// an oversight.
type Mode int

const (
	// ModePermissive is the zero value and the documented default
	// (SPEC-PG §2.2): INDETERMINATE never affects the Verdict. Making the
	// default itself the zero value is safe here -- unlike Outcome, where
	// a zero value that looked like success would hide a blind spot,
	// permissive mode does not hide INDETERMINATE results, it only
	// declines to let them deny the request by themselves. Reporting them
	// separately is the responsibility of whatever calls ComputeAggregate
	// and inspects the underlying []Result, not of this type.
	ModePermissive Mode = iota
	// ModeStrict denies the request when any block-severity check returns
	// INDETERMINATE, with Aggregate.Reason set to
	// DenyReasonInsufficientEvidence (SPEC-PG §2.2). It is the correct
	// setting for financial and irreversible operations, and is not the
	// default because a product that denies heavily on day one -- before
	// a customer's state supply is complete -- gets uninstalled on day
	// two.
	ModeStrict
)

// String renders the mode using the vocabulary from SPEC-PG §2.2.
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
