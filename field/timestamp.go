package field

import (
	"errors"
	"fmt"
)

// Encoding names how a timestamp field's value is represented on the wire
// (SPEC-MU MU-10 timestamp_encoding).
type Encoding int

const (
	// EncodingUnspecified is the zero value: no encoding was declared.
	EncodingUnspecified Encoding = iota
	// EncodingISO8601 declares an RFC 3339 string value.
	EncodingISO8601
	// EncodingEpochSeconds declares a Unix epoch value in seconds.
	EncodingEpochSeconds
	// EncodingEpochMillis declares a Unix epoch value in milliseconds.
	EncodingEpochMillis
)

// String renders the encoding using the vocabulary SPEC-MU §2.3's YAML
// example uses.
func (e Encoding) String() string {
	switch e {
	case EncodingISO8601:
		return "iso8601"
	case EncodingEpochSeconds:
		return "epoch_s"
	case EncodingEpochMillis:
		return "epoch_ms"
	default:
		return "unspecified"
	}
}

func (e Encoding) valid() bool {
	return e == EncodingISO8601 || e == EncodingEpochSeconds || e == EncodingEpochMillis
}

// TimestampDeclaration is the field declaration for `kind: timestamp`
// (SPEC-MU MU-10, MU-11). Its zero value, produced by
// NewTimestampDeclaration, declares nothing beyond the kind itself.
//
// NotBefore and NotAfter are stored as the raw declared string rather than
// a parsed instant: SPEC-MU MU-11 permits both absolute values and values
// relative to the request's evaluation timestamp ("now-5y", "now+90d"),
// and resolving "now" requires the evaluation timestamp that only exists
// at check time (task 1-6), not at declaration time.
type TimestampDeclaration struct {
	common

	encoding    Encoding
	hasEncoding bool

	notBefore    string
	hasNotBefore bool

	notAfter    string
	hasNotAfter bool
}

// NewTimestampDeclaration returns a TimestampDeclaration with no
// attributes declared beyond kind: timestamp. Chain With* methods to
// declare attributes.
func NewTimestampDeclaration() TimestampDeclaration { return TimestampDeclaration{} }

// Kind implements Declaration.
func (d TimestampDeclaration) Kind() Kind { return KindTimestamp }

// Encoding returns the declared wire encoding, if any. MU-10 returns
// INDETERMINATE when the second return value is false.
func (d TimestampDeclaration) Encoding() (Encoding, bool) { return d.encoding, d.hasEncoding }

// NotBefore returns the declared lower bound on this timestamp, if any, as
// the raw string the ruleset supplied (absolute, e.g. "2020-01-01T00:00:00Z",
// or relative, e.g. "now-5y").
func (d TimestampDeclaration) NotBefore() (string, bool) { return d.notBefore, d.hasNotBefore }

// NotAfter returns the declared upper bound on this timestamp, if any, in
// the same raw form as NotBefore.
func (d TimestampDeclaration) NotAfter() (string, bool) { return d.notAfter, d.hasNotAfter }

// WithEncoding declares the field's wire encoding. e must be
// EncodingISO8601, EncodingEpochSeconds, or EncodingEpochMillis.
func (d TimestampDeclaration) WithEncoding(e Encoding) (TimestampDeclaration, error) {
	if !e.valid() {
		return TimestampDeclaration{}, fmt.Errorf("field: invalid encoding %v", e)
	}
	d.encoding = e
	d.hasEncoding = true
	return d, nil
}

// WithNotBefore declares the field's lower bound, absolute or relative to
// evaluation time. bound must be non-empty.
func (d TimestampDeclaration) WithNotBefore(bound string) (TimestampDeclaration, error) {
	if bound == "" {
		return TimestampDeclaration{}, errors.New("field: not_before must not be empty")
	}
	d.notBefore = bound
	d.hasNotBefore = true
	return d, nil
}

// WithNotAfter declares the field's upper bound, absolute or relative to
// evaluation time. bound must be non-empty.
func (d TimestampDeclaration) WithNotAfter(bound string) (TimestampDeclaration, error) {
	if bound == "" {
		return TimestampDeclaration{}, errors.New("field: not_after must not be empty")
	}
	d.notAfter = bound
	d.hasNotAfter = true
	return d, nil
}

// WithNullSemantics declares the field's null-vs-absent handling (SPEC-MU
// MU-08). n must be NullSemanticsDistinct.
func (d TimestampDeclaration) WithNullSemantics(n NullSemantics) (TimestampDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return TimestampDeclaration{}, err
	}
	d.common = c
	return d, nil
}
