package tables

import (
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
)

// unitRegistryVersion is this package's own semver for the curated unit
// data below (see the Versioned doc comment: "semver for a unit
// registry"). Unlike CurrencyTable/CountryTable, this table is not
// generated from an external authority -- there is no single canonical
// source for "the unit registry" the way ISO 4217 and ISO 3166 are
// canonical sources for currency and country codes -- so it is hand
// curated here and versioned like any other piece of source under this
// module's own release discipline.
const unitRegistryVersion = "1.0.0"

// mustDecimal parses s as an exact decimal, panicking if s is not valid
// decimal text. Every call site below passes a fixed literal this
// package's own test suite has verified parses -- see
// TestMustDecimal_PanicsOnInvalidInput for the panic branch itself, which
// no real call site here can reach. Mirrors mu.mustParseDecimal's
// same-shaped, deliberately unexported helper; see its doc comment for why
// this is not folded into the decimal package itself.
func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("tables: mustDecimal(%q): %v", s, err))
	}
	return d
}

// zero is the exact decimal 0, used throughout unitRows for the (overwhelming
// majority of) units whose conversion is purely multiplicative -- no
// affine offset at all. Only temperature (°C, °F) needs a non-zero
// ToOffset/FromOffset.
func zero() decimal.Decimal { return mustDecimal("0") }

// one is the exact decimal 1, used for every dimension's own canonical
// unit (kg, m, K, ...), whose ToCanonical/FromCanonical are both the
// identity transform.
func one() decimal.Decimal { return mustDecimal("1") }

// unitRows returns this package's compiled-in unit registry data: one Unit
// per registered symbol, covering the twelve dimensions SPEC-MU §4 MU-04
// enumerates. It allocates a fresh slice on every call -- see
// NewUnitRegistry's own doc comment for why (this package's "no
// package-level state" rule).
//
// # Where each dimension's constants came from, and why some are truncated
//
// Every scale factor below is either an internationally defined exact
// decimal (a pound is *defined* as exactly 0.45359237 kg; a US gallon as
// exactly 3.785411784 L; and so on for the other SI-adjacent conversions
// in common use) or, where the exact value or its reciprocal is not a
// terminating decimal at all (kg → lb, s → min, m/s → km/h, any conversion
// touching π for angle, Fahrenheit's 5/9), a decimal literal truncated --
// never rounded -- to 20 fractional digits. SPEC-MU §4 MU-15's own text is
// explicit that this is unavoidable, not a shortcut this registry is
// taking: "A conversion factor that is not exactly representable as a
// decimal cannot round-trip in the value model of section 2.6.1... every
// implementation must truncate it somewhere. Where it truncates is its own
// business." Twenty digits is comfortably beyond MU-15's default relative
// tolerance of 1 part in 10^9 for every dimension except the one SPEC-MU
// names as the deliberate exception: Fahrenheit, whose forward conversion
// (°F → K) uses the truncated 5/9 and therefore does not round-trip even
// at that tolerance when declared with `tolerance: "0"` (vector 97) --
// while its reverse (K → °F) uses 9/5 = 1.8 and -459.67, both exact.
//
// # Two independent affine directions, not one inverted
//
// Every unit's four constants (ToScale/ToOffset, FromScale/FromOffset) are
// supplied as they appear below, never derived from one another at
// runtime -- decimal has no division operator, and Unit's own doc comment
// explains why that is a requirement here, not a limitation this file
// works around.
//
// # currency_per_unit: a deliberately narrow illustration, not full coverage
//
// SPEC-MU §4 names "currency-per-unit" as a supported dimension without
// further defining its unit space, which is open-ended in a way none of
// the other eleven dimensions are: any ISO 4217 code combined with any
// base quantity's unit is nominally a valid currency-per-unit symbol, and
// converting between two of them is only mathematically meaningful when
// both name the *same* currency and the *same underlying physical
// dimension* (a price per kilogram converts to a price per gram; it does
// not convert to a price per litre without an unknowable density, and it
// does not convert to a different currency without a rate this pure,
// I/O-free engine has no way to obtain -- SPEC-SYS §14.1). This registry
// therefore implements one small, internally coherent family --
// USD-denominated prices per unit mass -- as a working illustration of the
// dimension SPEC-MU names, not a claim of general currency-per-unit
// coverage. No SPEC-MU §8.3 conformance vector exercises this dimension.
func unitRows() []Unit {
	return []Unit{
		// Mass (canonical: kg).
		mustUnit("kg", DimensionMass, one(), zero(), one(), zero()),
		mustUnit("g", DimensionMass, mustDecimal("0.001"), zero(), mustDecimal("1000"), zero()),
		mustUnit("mg", DimensionMass, mustDecimal("0.000001"), zero(), mustDecimal("1000000"), zero()),
		mustUnit("lb", DimensionMass, mustDecimal("0.45359237"), zero(), mustDecimal("2.20462262184877580722"), zero()),
		mustUnit("oz", DimensionMass, mustDecimal("0.028349523125"), zero(), mustDecimal("35.27396194958041291567"), zero()),

		// Length (canonical: m).
		mustUnit("m", DimensionLength, one(), zero(), one(), zero()),
		mustUnit("km", DimensionLength, mustDecimal("1000"), zero(), mustDecimal("0.001"), zero()),
		mustUnit("cm", DimensionLength, mustDecimal("0.01"), zero(), mustDecimal("100"), zero()),
		mustUnit("mm", DimensionLength, mustDecimal("0.001"), zero(), mustDecimal("1000"), zero()),
		mustUnit("mi", DimensionLength, mustDecimal("1609.344"), zero(), mustDecimal("0.00062137119223733396"), zero()),
		mustUnit("yd", DimensionLength, mustDecimal("0.9144"), zero(), mustDecimal("1.09361329833770778652"), zero()),
		mustUnit("ft", DimensionLength, mustDecimal("0.3048"), zero(), mustDecimal("3.28083989501312335958"), zero()),
		mustUnit("in", DimensionLength, mustDecimal("0.0254"), zero(), mustDecimal("39.37007874015748031496"), zero()),

		// Volume (canonical: L).
		mustUnit("L", DimensionVolume, one(), zero(), one(), zero()),
		mustUnit("mL", DimensionVolume, mustDecimal("0.001"), zero(), mustDecimal("1000"), zero()),
		mustUnit("m3", DimensionVolume, mustDecimal("1000"), zero(), mustDecimal("0.001"), zero()),
		mustUnit("gal", DimensionVolume, mustDecimal("3.785411784"), zero(), mustDecimal("0.26417205235814841537"), zero()),
		mustUnit("qt", DimensionVolume, mustDecimal("0.946352946"), zero(), mustDecimal("1.05668820943259366151"), zero()),
		mustUnit("pt", DimensionVolume, mustDecimal("0.473176473"), zero(), mustDecimal("2.11337641886518732303"), zero()),
		mustUnit("floz", DimensionVolume, mustDecimal("0.0295735295625"), zero(), mustDecimal("33.81402270184299716862"), zero()),

		// Time (canonical: s).
		mustUnit("s", DimensionTime, one(), zero(), one(), zero()),
		mustUnit("ms", DimensionTime, mustDecimal("0.001"), zero(), mustDecimal("1000"), zero()),
		mustUnit("min", DimensionTime, mustDecimal("60"), zero(), mustDecimal("0.01666666666666666666"), zero()),
		mustUnit("h", DimensionTime, mustDecimal("3600"), zero(), mustDecimal("0.00027777777777777777"), zero()),
		mustUnit("day", DimensionTime, mustDecimal("86400"), zero(), mustDecimal("0.00001157407407407407"), zero()),

		// Temperature (canonical: K). Affine -- see this function's doc
		// comment for why °F's constants are asymmetrically exact/truncated.
		mustUnit("K", DimensionTemperature, one(), zero(), one(), zero()),
		mustUnit("°C", DimensionTemperature, one(), mustDecimal("273.15"), one(), mustDecimal("-273.15")),
		mustUnit("°F", DimensionTemperature,
			mustDecimal("0.55555555555555555555"), mustDecimal("255.37222222222222222240"),
			mustDecimal("1.8"), mustDecimal("-459.67")),

		// Area (canonical: m2).
		mustUnit("m2", DimensionArea, one(), zero(), one(), zero()),
		mustUnit("km2", DimensionArea, mustDecimal("1000000"), zero(), mustDecimal("0.000001"), zero()),
		mustUnit("cm2", DimensionArea, mustDecimal("0.0001"), zero(), mustDecimal("10000"), zero()),
		mustUnit("ft2", DimensionArea, mustDecimal("0.09290304"), zero(), mustDecimal("10.76391041670972230833"), zero()),
		mustUnit("acre", DimensionArea, mustDecimal("4046.8564224"), zero(), mustDecimal("0.00024710538146716534"), zero()),
		mustUnit("ha", DimensionArea, mustDecimal("10000"), zero(), mustDecimal("0.0001"), zero()),

		// Speed (canonical: m/s).
		mustUnit("m/s", DimensionSpeed, one(), zero(), one(), zero()),
		mustUnit("km/h", DimensionSpeed, mustDecimal("0.27777777777777777777"), zero(), mustDecimal("3.6"), zero()),
		mustUnit("mph", DimensionSpeed, mustDecimal("0.44704"), zero(), mustDecimal("2.23693629205440229062"), zero()),

		// Energy (canonical: J).
		mustUnit("J", DimensionEnergy, one(), zero(), one(), zero()),
		mustUnit("kJ", DimensionEnergy, mustDecimal("1000"), zero(), mustDecimal("0.001"), zero()),
		mustUnit("cal", DimensionEnergy, mustDecimal("4.184"), zero(), mustDecimal("0.23900573613766730401"), zero()),
		mustUnit("kcal", DimensionEnergy, mustDecimal("4184"), zero(), mustDecimal("0.00023900573613766730"), zero()),
		mustUnit("kWh", DimensionEnergy, mustDecimal("3600000"), zero(), mustDecimal("0.00000027777777777777"), zero()),

		// Pressure (canonical: Pa).
		mustUnit("Pa", DimensionPressure, one(), zero(), one(), zero()),
		mustUnit("kPa", DimensionPressure, mustDecimal("1000"), zero(), mustDecimal("0.001"), zero()),
		mustUnit("bar", DimensionPressure, mustDecimal("100000"), zero(), mustDecimal("0.00001"), zero()),
		mustUnit("atm", DimensionPressure, mustDecimal("101325"), zero(), mustDecimal("0.00000986923266716012"), zero()),
		mustUnit("psi", DimensionPressure, mustDecimal("6894.75729316836133672267"), zero(), mustDecimal("0.00014503773773020921"), zero()),

		// Digital storage (canonical: B). Decimal (1000-based) multiples --
		// a deliberate, documented convention choice; this registry does
		// not also carry binary (1024-based) KiB/MiB/GiB/TiB variants.
		mustUnit("B", DimensionDigitalStorage, one(), zero(), one(), zero()),
		mustUnit("KB", DimensionDigitalStorage, mustDecimal("1000"), zero(), mustDecimal("0.001"), zero()),
		mustUnit("MB", DimensionDigitalStorage, mustDecimal("1000000"), zero(), mustDecimal("0.000001"), zero()),
		mustUnit("GB", DimensionDigitalStorage, mustDecimal("1000000000"), zero(), mustDecimal("0.000000001"), zero()),
		mustUnit("TB", DimensionDigitalStorage, mustDecimal("1000000000000"), zero(), mustDecimal("0.000000000001"), zero()),

		// Angle (canonical: rad).
		mustUnit("rad", DimensionAngle, one(), zero(), one(), zero()),
		mustUnit("deg", DimensionAngle, mustDecimal("0.01745329251994329576"), zero(), mustDecimal("57.29577951308232087679"), zero()),
		mustUnit("grad", DimensionAngle, mustDecimal("0.01570796326794896619"), zero(), mustDecimal("63.66197723675813430755"), zero()),

		// Currency-per-unit (canonical: USD/kg) -- see this function's doc
		// comment for the deliberately narrow scope.
		mustUnit("USD/kg", DimensionCurrencyPerUnit, one(), zero(), one(), zero()),
		mustUnit("USD/g", DimensionCurrencyPerUnit, mustDecimal("1000"), zero(), mustDecimal("0.001"), zero()),
		mustUnit("USD/lb", DimensionCurrencyPerUnit, mustDecimal("2.20462262184877580722"), zero(), mustDecimal("0.45359237"), zero()),
	}
}
