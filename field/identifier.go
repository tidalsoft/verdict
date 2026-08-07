package field

import "errors"

// Scheme names an identifier checksum or membership algorithm (SPEC-MU
// MU-16 identifier_checksum). It is a plain string type, not a closed set
// this package enforces: MU-16 requires an *unrecognised* scheme to
// evaluate as INDETERMINATE, never to be rejected before MU-16 ever runs,
// so this package must not narrow what a ruleset is allowed to declare
// here. The constants below name the schemes SPEC-MU §5 defines; a
// ruleset may still declare a string that matches none of them, which is
// exactly the "unrecognised" case MU-16 (task 1-6) must handle.
type Scheme string

const (
	// SchemeIBAN validates ISO 13616 mod-97-10.
	SchemeIBAN Scheme = "iban"
	// SchemeLuhn validates a payment card number (ISO/IEC 7812).
	SchemeLuhn Scheme = "luhn"
	// SchemeISBN13 validates a 13-digit ISBN's weighted modulus.
	SchemeISBN13 Scheme = "isbn13"
	// SchemeISBN10 validates a 10-digit ISBN's weighted modulus.
	SchemeISBN10 Scheme = "isbn10"
	// SchemeGTIN8 validates an 8-digit GS1 modulus-10 check digit.
	SchemeGTIN8 Scheme = "gtin8"
	// SchemeGTIN12 validates a 12-digit GS1 modulus-10 check digit.
	SchemeGTIN12 Scheme = "gtin12"
	// SchemeGTIN13 validates a 13-digit GS1 modulus-10 check digit.
	SchemeGTIN13 Scheme = "gtin13"
	// SchemeGTIN14 validates a 14-digit GS1 modulus-10 check digit.
	SchemeGTIN14 Scheme = "gtin14"
	// SchemeLEI validates ISO 17442 mod-97-10.
	SchemeLEI Scheme = "lei"
	// SchemeBIC validates ISO 9362 structure (no check digit).
	SchemeBIC Scheme = "bic"
	// SchemeISO4217 validates membership in the versioned ISO 4217
	// currency table (verdict/tables.CurrencyTable).
	SchemeISO4217 Scheme = "iso4217"
	// SchemeISO3166Alpha2 validates membership in the versioned ISO
	// 3166-1 alpha-2 country table (verdict/tables.CountryTable).
	SchemeISO3166Alpha2 Scheme = "iso3166_alpha2"
)

// IdentifierDeclaration is the field declaration for `kind: identifier`
// (SPEC-MU MU-16). Its zero value, produced by NewIdentifierDeclaration,
// declares nothing beyond the kind itself.
type IdentifierDeclaration struct {
	common

	scheme    Scheme
	hasScheme bool
}

// NewIdentifierDeclaration returns an IdentifierDeclaration with no
// attributes declared beyond kind: identifier. Chain With* methods to
// declare attributes.
func NewIdentifierDeclaration() IdentifierDeclaration { return IdentifierDeclaration{} }

// Kind implements Declaration.
func (d IdentifierDeclaration) Kind() Kind { return KindIdentifier }

// Scheme returns the declared scheme, if any. MU-16 returns INDETERMINATE
// when the second return value is false -- a field declared `kind:
// identifier` with no scheme at all cannot be validated any more than one
// whose scheme string matches nothing this package or MU-16 recognises.
func (d IdentifierDeclaration) Scheme() (Scheme, bool) { return d.scheme, d.hasScheme }

// WithScheme declares the field's identifier scheme. s must be non-empty;
// it need not be one of the named Scheme constants -- see the Scheme doc
// comment for why an unrecognised value is accepted here and left for
// MU-16 to report as INDETERMINATE.
func (d IdentifierDeclaration) WithScheme(s Scheme) (IdentifierDeclaration, error) {
	if s == "" {
		return IdentifierDeclaration{}, errors.New("field: scheme must not be empty")
	}
	d.scheme = s
	d.hasScheme = true
	return d, nil
}

// WithNullSemantics declares the field's null-vs-absent handling (SPEC-MU
// MU-08). n must be NullSemanticsDistinct.
func (d IdentifierDeclaration) WithNullSemantics(n NullSemantics) (IdentifierDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return IdentifierDeclaration{}, err
	}
	d.common = c
	return d, nil
}
