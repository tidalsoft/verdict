package field

import "github.com/tidalsoft/verdict/decimal"

// ValueKind identifies which JSON shape a Value carries.
//
// SPEC-MU §3 MU-06 (sign_convention) partitions every value a `when` path
// can resolve to, or a `sign_when` entry can state, into two buckets: four
// **comparable** shapes -- a number, a string, a boolean, or `null` -- and
// everything else, which SPEC-MU calls "a sequence, another JSON array, or
// a JSON object." Only the four comparable shapes participate in equality
// at all; the rest fold into ValueKindNonComparable, because this package
// never needs to distinguish *which* non-comparable shape a value was --
// SPEC-MU §3 treats a sequence (a `[*]`-wildcard path, section 2.4.2), a
// plain array, and an object identically, as three names for one outcome.
type ValueKind int

const (
	// ValueKindUnspecified is the zero value. It is never produced by any
	// constructor in this file, so a zero-initialized Value (an
	// uninitialized struct field, a map lookup miss handled incorrectly)
	// is visibly not one of the five real kinds -- Comparable() reports
	// false for it, matching the safe reading "not a value at all."
	ValueKindUnspecified ValueKind = iota
	// ValueKindString marks a value that resolved to (or was declared as)
	// a JSON string.
	ValueKindString
	// ValueKindNumber marks a value that resolved to (or was declared as)
	// a JSON number, held as an exact decimal (SPEC-MU §2.6.1: never
	// float64).
	ValueKindNumber
	// ValueKindBool marks a value that resolved to (or was declared as) a
	// JSON boolean.
	ValueKindBool
	// ValueKindNull marks an explicit JSON `null` -- distinct from a path
	// that did not resolve at all (comma-ok absence from a caller's
	// map[string]Value), exactly as MU-08 (null_vs_absent) requires null
	// and absence to be told apart (SPEC-MU §3 MU-06 vector 92: an
	// explicit `null` resolves, and matches only a clause stating `null`).
	ValueKindNull
	// ValueKindNonComparable marks a value that resolved to (or was
	// declared as) a sequence, a JSON array, or a JSON object -- shapes
	// SPEC-MU §3 excludes from every comparison in this document (vectors
	// 118, 122).
	ValueKindNonComparable
)

// String renders the kind's canonical name. An out-of-range value renders
// as "unspecified" rather than panicking.
func (k ValueKind) String() string {
	switch k {
	case ValueKindString:
		return "string"
	case ValueKindNumber:
		return "number"
	case ValueKindBool:
		return "bool"
	case ValueKindNull:
		return "null"
	case ValueKindNonComparable:
		return "non_comparable"
	default:
		return "unspecified"
	}
}

// Value is a single JSON-shaped value: a sibling field a check resolves
// through a request (MU-03's currency sibling, MU-06's `sign_when`
// condition, MU-04/MU-05/MU-15's `unit_field`), or the stated value a
// ruleset declares to compare it against (a `sign_when` clause's `when`
// map). One type serves both directions so the "comparable shapes" test
// (Equal, Comparable) is implemented exactly once for whichever package
// needs it -- this one, for declared values, and mu, for resolved ones --
// rather than risking two implementations quietly disagreeing at the
// edges.
//
// The zero Value is ValueKindUnspecified, not comparable to anything,
// including another zero Value. Every real Value is built by one of this
// file's constructors.
type Value struct {
	kind ValueKind
	str  string
	num  decimal.Decimal
	b    bool
}

// NewStringValue returns a Value holding a JSON string.
func NewStringValue(s string) Value { return Value{kind: ValueKindString, str: s} }

// NewNumberValue returns a Value holding a JSON number, as an exact
// decimal -- SPEC-MU §2.6.1 forbids float64 anywhere in the evaluation
// path, including here.
func NewNumberValue(n decimal.Decimal) Value { return Value{kind: ValueKindNumber, num: n} }

// NewBoolValue returns a Value holding a JSON boolean.
func NewBoolValue(b bool) Value { return Value{kind: ValueKindBool, b: b} }

// NewNullValue returns a Value holding an explicit JSON `null` -- distinct
// from a path that simply did not resolve (see the ValueKindNull doc
// comment).
func NewNullValue() Value { return Value{kind: ValueKindNull} }

// NewNonComparableValue returns a Value marking a resolved or declared
// shape SPEC-MU §3 excludes from comparison: a sequence, a JSON array, or
// a JSON object. This package carries no representation of the value
// itself in that case -- callers holding one need only know that it does
// not compare, never what it was.
func NewNonComparableValue() Value { return Value{kind: ValueKindNonComparable} }

// Kind reports which of the five ValueKind values v holds.
func (v Value) Kind() ValueKind { return v.kind }

// StringValue returns v's string payload and whether v is
// ValueKindString.
func (v Value) StringValue() (string, bool) { return v.str, v.kind == ValueKindString }

// NumberValue returns v's decimal payload and whether v is
// ValueKindNumber.
func (v Value) NumberValue() (decimal.Decimal, bool) { return v.num, v.kind == ValueKindNumber }

// BoolValue returns v's boolean payload and whether v is ValueKindBool.
func (v Value) BoolValue() (bool, bool) { return v.b, v.kind == ValueKindBool }

// IsNull reports whether v is an explicit JSON null.
func (v Value) IsNull() bool { return v.kind == ValueKindNull }

// Comparable reports whether v is one of SPEC-MU §3's four comparable
// shapes (number, string, boolean, null). A ValueKindNonComparable value,
// and the zero Value (ValueKindUnspecified), both report false.
func (v Value) Comparable() bool {
	switch v.kind {
	case ValueKindString, ValueKindNumber, ValueKindBool, ValueKindNull:
		return true
	default:
		return false
	}
}

// Equal reports whether v and other are the same comparable JSON value --
// SPEC-MU §3 MU-06: values are "compared as JSON values, by type and
// content together, and only where both are one of four comparable
// shapes... Numbers compare by numeric value, so a declared 500 matches a
// resolved 500.0[;] strings compare as exact sequences of codepoints, with
// no case folding and no trimming; booleans and null compare only with
// themselves." A declared 500 does not match a resolved "500" -- Equal
// requires both the kind and the content to agree, never coercing across
// JSON types.
//
// Equal returns false whenever either side is not Comparable(), which
// conflates "compared and disagreed" with "did not compare at all" into
// one boolean. MU-06's three-state clause taxonomy (matches / undecidable
// / definitively does not match, SPEC-MU §3) needs that distinction, so a
// caller building it must call Comparable() on both sides before relying
// on Equal to mean "disagreed" -- see mu.clauseState (sign.go), the one
// caller in this module that does.
func (v Value) Equal(other Value) bool {
	if !v.Comparable() || !other.Comparable() || v.kind != other.kind {
		return false
	}
	switch v.kind {
	case ValueKindString:
		return v.str == other.str
	case ValueKindNumber:
		return v.num.Compare(other.num) == 0
	case ValueKindBool:
		return v.b == other.b
	default:
		// The guard above already restricts v.kind to the four Comparable
		// shapes; String, Number, and Bool are handled above, so the only
		// value that reaches here is ValueKindNull, and null equals null
		// unconditionally (SPEC-MU §3: "null[s]... compare only with
		// themselves").
		return true
	}
}
