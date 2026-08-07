//go:build ignore

// Command iso4217gen regenerates tables/currency_data.go from the
// ISO 4217 "Current currency & funds code list" (Table A.1), the primary
// source for ISO 4217 currency codes and minor-unit exponents (SPEC-MU
// MU-14). The ISO 4217 Maintenance Agency delegates distribution of this
// list, in machine-readable XML, to SIX Group:
//
//	https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list-one.xml
//
// The exact file fetched on 2026-08-07 is committed alongside this program
// as list-one.xml, so regeneration against the same input is reproducible
// without a further network fetch, and a future re-fetch's diff against it
// shows exactly what the Maintenance Agency changed. To pull a fresh copy
// and regenerate against it, from the repository root:
//
//	curl -o tables/generate/iso4217/list-one.xml \
//	  https://www.six-group.com/dam/download/financial-information/data-center/iso-currrency/lists/list-one.xml
//	go run tables/generate/iso4217/main.go
//
// This program is excluded from the module's ordinary build, vet, lint, and
// coverage runs by the go:build tag above: it is a build-time source
// generator that reads a file and writes a file, which the pure evaluation
// engine it feeds (CLAUDE.md invariant #3 -- no filesystem access in
// verdict/) must never do. Run it explicitly with `go run`, not `go build
// ./...` or `go test ./...`.
package main

import (
	"encoding/xml"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	sourcePath = "tables/generate/iso4217/list-one.xml"
	outputPath = "tables/currency_data.go"
)

// isoTable mirrors the subset of list-one.xml's structure this program
// needs. The Pblshd attribute is the Maintenance Agency's own publication
// date for the list, which becomes this table's reported version
// (SPEC-MU §7.2, §9) -- not a date this program invents.
type isoTable struct {
	XMLName xml.Name   `xml:"ISO_4217"`
	Pblshd  string     `xml:"Pblshd,attr"`
	Entries []ccyEntry `xml:"CcyTbl>CcyNtry"`
}

type ccyEntry struct {
	Ccy        string `xml:"Ccy"`
	CcyMnrUnts string `xml:"CcyMnrUnts"`
}

type row struct {
	code        string
	exponent    int32
	hasExponent bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "iso4217gen:", err)
		os.Exit(1)
	}
}

func run() error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", sourcePath, err)
	}

	var doc isoTable
	if err := xml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", sourcePath, err)
	}
	if doc.Pblshd == "" {
		return fmt.Errorf("%s: missing Pblshd attribute on root element", sourcePath)
	}
	version, err := tableVersion(doc.Pblshd)
	if err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}

	rows, err := collectRows(doc.Entries)
	if err != nil {
		return err
	}

	src, err := renderSource(doc.Pblshd, version, rows)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}
	fmt.Printf("iso4217gen: wrote %s (%d currency/fund codes, version %s)\n", outputPath, len(rows), version)
	return nil
}

// tableVersion derives the "YYYY-MM" version SPEC-MU §7.2's example uses
// (`"iso4217": "2026-01"`) from the XML's full "YYYY-MM-DD" Pblshd date.
func tableVersion(pblshd string) (string, error) {
	if len(pblshd) < 7 {
		return "", fmt.Errorf("Pblshd %q is too short to derive a YYYY-MM version", pblshd)
	}
	return pblshd[:7], nil
}

// collectRows reduces the raw XML entries (one per currency-issuing
// country/territory, so most currencies appear many times -- EUR appears
// once per Eurozone member) to one row per distinct alphabetic code,
// failing loudly if the same code disagrees with itself about its minor
// unit anywhere in the source, since that would mean this program's
// row-per-code assumption no longer holds.
func collectRows(entries []ccyEntry) ([]row, error) {
	seen := make(map[string]row)
	for _, e := range entries {
		code := strings.TrimSpace(e.Ccy)
		if code == "" {
			continue // entries with no Ccy element exist in list-three.xml's shape, not list-one.xml, but skip defensively
		}
		r := row{code: code}
		mn := strings.TrimSpace(e.CcyMnrUnts)
		if mn != "" && mn != "N.A." {
			n, err := strconv.Atoi(mn)
			if err != nil {
				return nil, fmt.Errorf("currency %s: minor unit %q is neither empty, N.A., nor an integer: %w", code, mn, err)
			}
			r.exponent = int32(n)
			r.hasExponent = true
		}
		if prior, ok := seen[code]; ok {
			if prior != r {
				return nil, fmt.Errorf("currency %s: source disagrees with itself (%+v vs %+v)", code, prior, r)
			}
			continue
		}
		seen[code] = r
	}

	codes := make([]string, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	rows := make([]row, 0, len(codes))
	for _, code := range codes {
		rows = append(rows, seen[code])
	}
	return rows, nil
}

func renderSource(pblshd, version string, rows []row) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by `go run tables/generate/iso4217/main.go`. DO NOT EDIT.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Source: ISO 4217 Table A.1, \"Current currency & funds code list\",\n")
	fmt.Fprintf(&b, "// published %s by the ISO 4217 Maintenance Agency, distributed as XML by\n", pblshd)
	fmt.Fprintf(&b, "// SIX Group. Committed snapshot: tables/generate/iso4217/list-one.xml.\n")
	fmt.Fprintf(&b, "// %d distinct currency and fund codes.\n", len(rows))
	fmt.Fprintf(&b, "package tables\n\n")
	fmt.Fprintf(&b, "// iso4217Version is ISO 4217 Table A.1's own publication date (the XML\n")
	fmt.Fprintf(&b, "// root element's Pblshd attribute, %q), truncated to the \"YYYY-MM\" form\n", pblshd)
	fmt.Fprintf(&b, "// SPEC-MU §7.2's example response uses.\n")
	fmt.Fprintf(&b, "const iso4217Version = %q\n\n", version)
	fmt.Fprintf(&b, "// iso4217Rows returns the compiled-in ISO 4217 currency & funds code list.\n")
	fmt.Fprintf(&b, "// It returns a fresh slice on every call rather than sharing one across\n")
	fmt.Fprintf(&b, "// calls, in keeping with this package holding no package-level state;\n")
	fmt.Fprintf(&b, "// NewISO4217Table is the only intended caller. It builds Currency values\n")
	fmt.Fprintf(&b, "// directly (rather than an intermediate row type) using this package's\n")
	fmt.Fprintf(&b, "// unexported fields, since this file and currency.go share a package.\n")
	fmt.Fprintf(&b, "func iso4217Rows() []Currency {\n")
	fmt.Fprintf(&b, "\treturn []Currency{\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "\t\t{code: %q, minorUnitExponent: %d, hasMinorUnitExponent: %t},\n", r.code, r.exponent, r.hasExponent)
	}
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "}\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt generated source: %w", err)
	}
	return formatted, nil
}
