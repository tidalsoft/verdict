package field

// Kind identifies which field-declaration shape a Declaration
// carries. Each Kind has exactly one corresponding concrete type
// (MoneyDeclaration, QuantityDeclaration, TimestampDeclaration,
// PercentageDeclaration, DecimalDeclaration, IdentifierDeclaration); Kind
// is how code holding only the Declaration interface can decide which
// branch of a type switch to take, or simply report what kind a field was
// declared as, without an unchecked type assertion.
type Kind int

const (
	// KindUnspecified is the zero value. No concrete Declaration type ever
	// reports it from Kind() -- every constructor in this package produces
	// a value whose Kind() is fixed and non-zero -- so a zero-initialized
	// Kind (an uninitialized struct field, a map lookup miss) reads as
	// "not set" rather than silently aliasing KindMoney.
	KindUnspecified Kind = iota
	// KindMoney declares a field as a monetary amount: MoneyDeclaration.
	KindMoney
	// KindQuantity declares a field as a dimensioned physical quantity:
	// QuantityDeclaration.
	KindQuantity
	// KindTimestamp declares a field as a point in time (MU-10,
	// MU-11): TimestampDeclaration.
	KindTimestamp
	// KindPercentage declares a field as a fractional-or-hundred-scaled
	// ratio (MU-13): PercentageDeclaration.
	KindPercentage
	// KindDecimal declares a field as an exact decimal value that is not
	// itself money -- the escape hatch for intermediate calculation values
	// that legitimately carry more precision than a currency's minor unit
	// permits: DecimalDeclaration.
	KindDecimal
	// KindIdentifier declares a field as a structured identifier subject
	// to checksum or membership validation (MU-16): IdentifierDeclaration.
	KindIdentifier
)

// String renders the kind's canonical name. An out-of-range value
// (including the zero value, KindUnspecified) renders as "unspecified"
// rather than panicking.
func (k Kind) String() string {
	switch k {
	case KindMoney:
		return "money"
	case KindQuantity:
		return "quantity"
	case KindTimestamp:
		return "timestamp"
	case KindPercentage:
		return "percentage"
	case KindDecimal:
		return "decimal"
	case KindIdentifier:
		return "identifier"
	default:
		return "unspecified"
	}
}
