package verdict

import "fmt"

// Class distinguishes how a check's verdict is produced.
type Class int

const (
	// ClassUnspecified is the zero value. It is not a legitimate class for
	// an evaluated check -- NewResult rejects it -- and exists so that a
	// zero-initialized Class reads as "not set" rather than silently
	// aliasing ClassD or ClassS.
	ClassUnspecified Class = iota
	// ClassD (deterministic) checks compute their verdict as a pure
	// function of the argument and its declared schema: a FAIL means the
	// input contradicts its own declaration, so the false positive rate is
	// zero by construction. Class D defaults to SeverityBlock and may
	// block from the moment it ships.
	ClassD
	// ClassS (statistical) checks depend on a learned distribution or
	// heuristic and therefore carry a non-zero false positive rate. Class
	// S defaults to SeverityWarn and may only reach SeverityBlock for a
	// specific field after a measured, per-field promotion -- see
	// NewPromotedResult. This package models that restriction
	// structurally; the promotion lifecycle itself (fire counting,
	// precision measurement, and the minimum sample size required before
	// promotion) happens outside this library and is out of scope here.
	ClassS
)

// String renders the class as its canonical single-letter code. An
// out-of-range value (including the zero value, ClassUnspecified) renders
// as "UNKNOWN_CLASS" rather than panicking.
func (c Class) String() string {
	switch c {
	case ClassD:
		return "D"
	case ClassS:
		return "S"
	default:
		return "UNKNOWN_CLASS"
	}
}

// valid reports whether c is one of the two legitimate classes. The zero
// value, ClassUnspecified, is not valid -- see the Class doc comment.
// Unexported: it exists so NewResult can refuse an out-of-range or
// zero-value Class instead of silently treating it as ClassD or ClassS.
func (c Class) valid() bool {
	return c == ClassD || c == ClassS
}

// DefaultSeverity returns the severity a check of this class carries unless
// a ruleset explicitly overrides it. It returns an error for any class
// other than ClassD or ClassS.
func (c Class) DefaultSeverity() (Severity, error) {
	switch c {
	case ClassD:
		return SeverityBlock, nil
	case ClassS:
		return SeverityWarn, nil
	default:
		return SeverityUnspecified, fmt.Errorf("verdict: class %v has no default severity", c)
	}
}
