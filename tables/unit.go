package tables

import (
	"errors"
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
)

// Dimension identifies one of the physical dimensions SPEC-MU §4 MU-04
// (unit_dimension_mismatch) enumerates under "Supported dimensions": mass,
// length, volume, time, temperature, area, speed, energy, pressure,
// digital storage, angle, and currency-per-unit.
type Dimension int

const (
	// DimensionUnspecified is the zero value: not a dimension any Unit in
	// this package's registry ever carries.
	DimensionUnspecified Dimension = iota
	// DimensionMass is the mass dimension (canonical unit: kg).
	DimensionMass
	// DimensionLength is the length dimension (canonical unit: m).
	DimensionLength
	// DimensionVolume is the volume dimension (canonical unit: L).
	DimensionVolume
	// DimensionTime is the time dimension (canonical unit: s).
	DimensionTime
	// DimensionTemperature is the temperature dimension (canonical unit:
	// K), the one dimension this registry converts affinely rather than
	// purely multiplicatively -- see Unit's doc comment.
	DimensionTemperature
	// DimensionArea is the area dimension (canonical unit: m2).
	DimensionArea
	// DimensionSpeed is the speed dimension (canonical unit: m/s).
	DimensionSpeed
	// DimensionEnergy is the energy dimension (canonical unit: J).
	DimensionEnergy
	// DimensionPressure is the pressure dimension (canonical unit: Pa).
	DimensionPressure
	// DimensionDigitalStorage is the digital-storage dimension (canonical
	// unit: B, decimal/1000-based multiples -- see unit_data.go's doc
	// comment).
	DimensionDigitalStorage
	// DimensionAngle is the angle dimension (canonical unit: rad).
	DimensionAngle
	// DimensionCurrencyPerUnit is a rate dimension -- a monetary amount per
	// unit of some other quantity (e.g. a price per kilogram). SPEC-MU §4
	// names it without further defining it; see unit_data.go's doc comment
	// for the deliberately narrow set this registry implements under it.
	DimensionCurrencyPerUnit
)

// String renders the dimension's canonical name -- the token a ruleset's
// `dimension:` attribute (SPEC-MU §2.4.2) is expected to spell, and what a
// declared field.QuantityDeclaration.Dimension() is compared against
// (mu.checkMU04). An out-of-range value (including the zero value)
// renders as "unspecified" rather than panicking.
func (d Dimension) String() string {
	switch d {
	case DimensionMass:
		return "mass"
	case DimensionLength:
		return "length"
	case DimensionVolume:
		return "volume"
	case DimensionTime:
		return "time"
	case DimensionTemperature:
		return "temperature"
	case DimensionArea:
		return "area"
	case DimensionSpeed:
		return "speed"
	case DimensionEnergy:
		return "energy"
	case DimensionPressure:
		return "pressure"
	case DimensionDigitalStorage:
		return "digital_storage"
	case DimensionAngle:
		return "angle"
	case DimensionCurrencyPerUnit:
		return "currency_per_unit"
	default:
		return "unspecified"
	}
}

func (d Dimension) valid() bool {
	switch d {
	case DimensionMass, DimensionLength, DimensionVolume, DimensionTime, DimensionTemperature,
		DimensionArea, DimensionSpeed, DimensionEnergy, DimensionPressure, DimensionDigitalStorage,
		DimensionAngle, DimensionCurrencyPerUnit:
		return true
	default:
		return false
	}
}

// Unit is a single entry in the unit registry: a symbol (e.g. "kg", "lb",
// "°F"), the Dimension it belongs to, and the affine transform to and from
// that dimension's chosen canonical unit -- canonical = value*ToScale +
// ToOffset, and the reverse, value = canonical*FromScale + FromOffset.
//
// The two directions are stored independently rather than one being
// derived from the other by division, for two reasons this package has no
// choice but to accept together. First, decimal (§2.6.1's value model) has
// no division operator at all -- only Add, Sub, Mul -- so there is no way
// to compute a reciprocal at construction time even if this package wanted
// to. Second, and more fundamentally, SPEC-MU §4 MU-15 requires that this
// be true regardless: "A conversion factor that is not exactly
// representable as a decimal cannot round-trip in the value model of
// section 2.6.1... Fahrenheit's scale factor is 5/9... every implementation
// must truncate it somewhere." Storing both directions as independently
// supplied, independently truncated decimal constants is what makes
// MU-15's round-trip check able to fail at all: a Unit computed by
// inverting one exact factor would round-trip perfectly by construction,
// and vector 97 -- 100°F, tolerance "0", FAIL -- would be unreachable by
// any conformant implementation of this registry.
type Unit struct {
	symbol     string
	dimension  Dimension
	toScale    decimal.Decimal
	toOffset   decimal.Decimal
	fromScale  decimal.Decimal
	fromOffset decimal.Decimal
}

// Symbol returns the unit's registry key (e.g. "kg").
func (u Unit) Symbol() string { return u.symbol }

// Dimension returns the physical dimension this unit belongs to.
func (u Unit) Dimension() Dimension { return u.dimension }

// ToCanonical converts value, expressed in u, to the dimension's canonical
// unit: value*ToScale + ToOffset, in exact decimal arithmetic throughout
// (SPEC-MU §2.6.1: never float64). Its only failure mode is the same one
// decimal.Decimal.Add/Mul document: the result's exponent overflowing the
// range this module's underlying arithmetic supports, which callers must
// treat as MU-07/MU-15's own INDETERMINATE (SPEC-MU §2.4's
// "does-not-abort-the-batch" rule), never as a panic or an aborted
// evaluation.
func (u Unit) ToCanonical(value decimal.Decimal) (decimal.Decimal, error) {
	scaled, err := value.Mul(u.toScale)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("tables: unit %q: convert to canonical: %w", u.symbol, err)
	}
	sum, err := scaled.Add(u.toOffset)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("tables: unit %q: convert to canonical: %w", u.symbol, err)
	}
	return sum, nil
}

// FromCanonical converts value, expressed in the dimension's canonical
// unit, to u: value*FromScale + FromOffset. See ToCanonical for the
// arithmetic and failure-mode contract; this is its mirror image, used by
// MU-15 to convert a canonical value back to a field's own unit for its
// round-trip comparison.
func (u Unit) FromCanonical(value decimal.Decimal) (decimal.Decimal, error) {
	scaled, err := value.Mul(u.fromScale)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("tables: unit %q: convert from canonical: %w", u.symbol, err)
	}
	sum, err := scaled.Add(u.fromOffset)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("tables: unit %q: convert from canonical: %w", u.symbol, err)
	}
	return sum, nil
}

// newUnit constructs a Unit, validating symbol and dimension the way every
// other constructor in this module validates its inputs (this package's
// "constructors validate everything" rule). It is unexported: the only
// caller is this package's own compiled-in data in unit_data.go, since --
// like CurrencyTable and CountryTable -- the engine performs no network or
// filesystem access at evaluation time and so cannot construct a Unit from
// anything but source compiled into the binary.
func newUnit(symbol string, dimension Dimension, toScale, toOffset, fromScale, fromOffset decimal.Decimal) (Unit, error) {
	if symbol == "" {
		return Unit{}, errors.New("tables: unit symbol must not be empty")
	}
	if !dimension.valid() {
		return Unit{}, fmt.Errorf("tables: unit %q: invalid dimension %v", symbol, dimension)
	}
	return Unit{
		symbol:     symbol,
		dimension:  dimension,
		toScale:    toScale,
		toOffset:   toOffset,
		fromScale:  fromScale,
		fromOffset: fromOffset,
	}, nil
}

// mustUnit calls newUnit and panics on error. Every call site in
// unit_data.go passes a fixed, valid literal -- the panic branch is
// unreachable from any of them -- so, per this module's established
// pattern for infallible-in-practice-but-tested constructors (see
// mu.mustResult/mu.mustParseDecimal), the error path is still real and
// still covered, just not from this function's own real callers; see
// TestMustUnit_PanicsOnInvalidInput.
func mustUnit(symbol string, dimension Dimension, toScale, toOffset, fromScale, fromOffset decimal.Decimal) Unit {
	u, err := newUnit(symbol, dimension, toScale, toOffset, fromScale, fromOffset)
	if err != nil {
		panic(fmt.Sprintf("tables: mustUnit(%q): %v", symbol, err))
	}
	return u
}

// UnitRegistry is an immutable, versioned unit-conversion table (SPEC-MU
// §4 MU-04/MU-05/MU-07/MU-15). Its zero value is not usable -- construct
// one with NewUnitRegistry. A UnitRegistry is safe for concurrent use,
// since nothing about it is ever mutated after NewUnitRegistry returns.
type UnitRegistry struct {
	version  string
	bySymbol map[string]Unit
}

// Version implements Versioned.
func (t UnitRegistry) Version() string { return t.version }

// Lookup returns the Unit registered under symbol, and whether one is.
// Matching is exact: this package performs no case-folding on unit
// symbols (unlike ISO 4217 currency codes, SPEC-MU §2.4.2 states no
// case-insensitivity rule for unit symbols, and folding "K" to "k" would
// silently confuse Kelvin with an SI kilo- prefix). A false second return
// value is what MU-04 treats as "unit not in the registry" (a FAIL, an
// unrecognised unit being unsafe to pass through) and what MU-05/MU-07/
// MU-15 treat as "no conversion factor" (INDETERMINATE) -- which of the
// two a false result means is each check's own decision, not this
// method's.
func (t UnitRegistry) Lookup(symbol string) (Unit, bool) {
	u, ok := t.bySymbol[symbol]
	return u, ok
}

// NewUnitRegistry builds the compiled-in unit registry. It is a pure
// function of this package's compiled-in data (unit_data.go) and
// allocates a lookup map on every call, so callers should build one
// registry once (e.g. alongside a ruleset or evaluator) and reuse it
// across evaluations, exactly as NewISO4217Table documents for
// CurrencyTable.
func NewUnitRegistry() UnitRegistry {
	units := unitRows()
	bySymbol := make(map[string]Unit, len(units))
	for _, u := range units {
		bySymbol[u.symbol] = u
	}
	return UnitRegistry{version: unitRegistryVersion, bySymbol: bySymbol}
}
