package verdict_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/tidalsoft/verdict-spec/vectors"
)

// TestConformanceVectorCoverage is the conformance gate: every vector
// SPEC-MU and SPEC-PG publish (github.com/tidalsoft/verdict-spec/vectors)
// must be either exercised by a test somewhere in this module, or named in
// skippedConformanceVectors below with a reason. An ID that is in neither
// set fails the build -- that is the whole point of this test: a vector
// with no test and no recorded skip must not be able to reach a green
// build silently.
//
// The "has a test" side is derived, never hand-maintained, by
// collectVectorIDs (see its doc comment for exactly which two AST shapes
// count). Comments are never consulted: go/parser is called with mode 0
// here, so comments are not even part of the parsed tree, and a comment
// mentioning a vector could not be mistaken for a test of it even if the
// scan were less careful than it is.
//
// Only files the Go toolchain would actually build are ever parsed.
// discoverImplementedVectorIDs asks go/build which files belong to each
// package, the same way `go build`/`go test` themselves decide, so a file
// under testdata/, under a "."- or "_"-prefixed directory, or excluded by
// its own //go:build line -- including a mistyped one -- is excluded here
// too, not silently read anyway.
//
// That view is the running machine's, because the build context depends on
// GOOS, GOARCH, and the active tags. A test behind //go:build linux counts
// on a linux machine and does not count on a mac. This is the property
// worth having rather than a flaw: the gate reports a vector implemented
// exactly when the machine running it would compile and run that test, so
// there is no machine where the gate is green and the test never executes.
// A platform-tagged conformance test therefore shows up as a local green
// and a CI red, and CI is the authority. Note also that -tags is not
// applied, so a test behind //go:build integration never counts here and
// still needs a skip entry.
//
// This gate proves a vector ID and some code coexist in the same file. It
// cannot prove that code actually exercises the vector's declaration,
// input, and expected outcome -- an empty `func TestFoo_MU_V70(t
// *testing.T) {}` satisfies it. That gap is deliberate and not something a
// static check can close; this gate exists to catch the far more common
// failure, a vector nobody wrote anything for at all, not to certify test
// quality.
func TestConformanceVectorCoverage(t *testing.T) {
	set, err := vectors.Load()
	if err != nil {
		t.Fatalf("vectors.Load: %v", err)
	}

	implemented := discoverImplementedVectorIDs(t)
	skipped := skippedConformanceVectors()

	allIDs := map[string]bool{}
	for _, v := range set.MU().Vectors {
		allIDs[v.ID] = true
	}
	for _, v := range set.PG().Vectors {
		allIDs[v.ID] = true
	}

	var (
		neither      []string // in neither implemented nor skipped
		both         []string // in both
		staleSkip    []string // skip entry names an ID no matrix defines
		unknownFound []string // implemented ID that no matrix defines
	)

	for id := range allIDs {
		_, isImpl := implemented[id]
		_, isSkip := skipped[id]
		switch {
		case isImpl && isSkip:
			both = append(both, id)
		case !isImpl && !isSkip:
			neither = append(neither, id)
		}
	}
	for id := range skipped {
		if !allIDs[id] {
			staleSkip = append(staleSkip, id)
		}
	}
	for id := range implemented {
		if !allIDs[id] {
			unknownFound = append(unknownFound, id)
		}
	}
	sort.Strings(neither)
	sort.Strings(both)
	sort.Strings(staleSkip)
	sort.Strings(unknownFound)

	if len(neither) > 0 || len(both) > 0 || len(staleSkip) > 0 || len(unknownFound) > 0 {
		var b strings.Builder
		if len(neither) > 0 {
			fmt.Fprintf(&b, "%d vector(s) have neither a test nor a recorded skip entry:\n", len(neither))
			for _, id := range neither {
				fmt.Fprintf(&b, "  %s\n", id)
			}
		}
		if len(both) > 0 {
			fmt.Fprintf(&b, "%d vector(s) are both claimed by a test and listed as skipped -- either the skip entry is stale and should be deleted, or the claim is wrong and should be investigated:\n", len(both))
			for _, id := range both {
				fmt.Fprintf(&b, "  %s (skip reason on file: %s)\n", id, skipped[id])
			}
		}
		if len(staleSkip) > 0 {
			fmt.Fprintf(&b, "%d skip entr(y/ies) name a vector ID no matrix defines:\n", len(staleSkip))
			for _, id := range staleSkip {
				fmt.Fprintf(&b, "  %s (skip reason: %s)\n", id, skipped[id])
			}
		}
		if len(unknownFound) > 0 {
			fmt.Fprintf(&b, "%d ID(s) found in test function names or vectorID field values that no matrix defines:\n", len(unknownFound))
			for _, id := range unknownFound {
				fmt.Fprintf(&b, "  %s\n", id)
			}
		}
		t.Fatal(b.String())
	}

	mu := set.MU()
	pg := set.PG()
	muImpl, pgImpl := 0, 0
	for _, v := range mu.Vectors {
		if implemented[v.ID] {
			muImpl++
		}
	}
	for _, v := range pg.Vectors {
		if implemented[v.ID] {
			pgImpl++
		}
	}
	t.Logf("SPEC-MU: %d vectors, %d implemented, %d skipped", len(mu.Vectors), muImpl, len(mu.Vectors)-muImpl)
	t.Logf("SPEC-PG: %d vectors, %d implemented, %d skipped", len(pg.Vectors), pgImpl, len(pg.Vectors)-pgImpl)
	t.Logf("total: %d vectors, %d implemented, %d skipped", len(allIDs), muImpl+pgImpl, len(allIDs)-muImpl-pgImpl)
}

// funcIDPattern and idPattern are package-level, unlike version_test.go's
// function-local semverRe, because both are compiled once and then reused
// across every file collectVectorIDs is called on for the duration of a
// single test run (once per _test.go file discoverImplementedVectorIDs's
// WalkDir callback visits, plus every call from
// TestCollectVectorIDs_OnlyFuncNamesAndVectorIDFields) -- compiling a fresh
// regexp.Regexp per file, rather than once per process, would be pure
// waste with no compensating benefit, since neither pattern depends on
// anything computed at run time.

// funcIDPattern matches a vector ID in the underscore form a test function
// name carries it in, e.g. "MU_V16" inside "TestCheckMU02_MU_V16".
var funcIDPattern = regexp.MustCompile(`(?:MU|PG)_V\d+`)

// idPattern matches a vector ID in its real, hyphenated form, e.g.
// "MU-V16". Used only to pull the ID back out of a vectorID field's string
// value (see collectVectorIDs) -- not run against arbitrary string
// literals, so a t.Log message or any other incidental string containing
// something that looks like an ID is never a match.
var idPattern = regexp.MustCompile(`(?:MU|PG)-V\d+`)

// collectVectorIDs walks file's AST and returns every vector ID it counts
// as implemented. Exactly two shapes count:
//
//   - a top-level function's name, e.g. "MU_V16" inside
//     "TestCheckMU02_MU_V16" (converted to its hyphenated form, "MU-V16")
//   - a string literal that is the value of a struct field named vectorID
//     in a keyed composite literal, e.g. {vectorID: "MU-V16", ...}
//
// Nothing else counts, deliberately. Earlier this also scanned every
// string literal in the file, which meant an incidental string anywhere --
// a t.Log message, a map literal's key, a doc comment's own reproduced
// text -- registered a vector as implemented whether or not anything
// tested it. Restricting collection to a named struct field closes that:
// a composite literal's positional elements and a map's string keys are
// not KeyValueExprs whose Key is an *ast.Ident, so they are structurally
// invisible here, not merely filtered out by convention.
func collectVectorIDs(file *ast.File) map[string]bool {
	found := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			for _, m := range funcIDPattern.FindAllString(node.Name.Name, -1) {
				found[strings.Replace(m, "_", "-", 1)] = true
			}
		case *ast.KeyValueExpr:
			key, ok := node.Key.(*ast.Ident)
			if !ok || key.Name != "vectorID" {
				return true
			}
			lit, ok := node.Value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				v = lit.Value
			}
			for _, m := range idPattern.FindAllString(v, -1) {
				found[m] = true
			}
		}
		return true
	})
	return found
}

// discoverImplementedVectorIDs walks every directory in this module
// starting from the directory this file itself lives in (found via
// runtime.Caller rather than the working directory, so this is correct
// regardless of which package's `go test` invocation is running it). It
// does not hardcode which package directories exist: mu, decimal, field,
// tables, and any directory added later are all discovered by the walk.
//
// The walk itself only decides which directories to look in, matching the
// Go toolchain's own package-discovery rules: a "testdata" directory, a
// "."-prefixed directory, or a "_"-prefixed directory is never treated as
// a package and its contents are never even listed, let alone parsed
// (plus "coverage", this repository's own build-artifact directory, which
// carries no Go source at all). Within a directory that is not skipped,
// which *files* actually belong to the package is decided by go/build, not
// by this function: build.ImportDir(dir, 0) applies the default build
// context -- the same one `go build`/`go test` (with no -tags) would use
// -- and returns exactly the file names the compiler would use, honouring
// every build constraint including a //go:build line, correctly typed or
// not. Only those files are parsed.
func discoverImplementedVectorIDs(t *testing.T) map[string]bool {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("discoverImplementedVectorIDs: runtime.Caller(0) failed to report this file's path")
	}
	root := filepath.Dir(thisFile)

	found := map[string]bool{}
	fset := token.NewFileSet()

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "coverage") {
			return filepath.SkipDir
		}

		pkg, ierr := build.ImportDir(path, 0)
		if ierr != nil {
			var noGo *build.NoGoError
			if errors.As(ierr, &noGo) {
				// No buildable Go source in this directory at
				// all (e.g. it holds only non-Go files, or
				// every file is excluded by build constraints).
				// That is an ordinary, expected shape for a
				// directory in this tree, not a failure.
				return nil
			}
			return fmt.Errorf("build.ImportDir(%s): %w", path, ierr)
		}

		testFiles := make([]string, 0, len(pkg.TestGoFiles)+len(pkg.XTestGoFiles))
		testFiles = append(testFiles, pkg.TestGoFiles...)
		testFiles = append(testFiles, pkg.XTestGoFiles...)
		for _, name := range testFiles {
			fpath := filepath.Join(path, name)
			// Mode 0: comments are not parsed into the tree at
			// all (see this file's top doc comment).
			file, perr := parser.ParseFile(fset, fpath, nil, 0)
			if perr != nil {
				return fmt.Errorf("parsing %s: %w", fpath, perr)
			}
			for id := range collectVectorIDs(file) {
				found[id] = true
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("discoverImplementedVectorIDs: walking module tree: %v", walkErr)
	}
	return found
}

// Skip reasons, one per missing capability. Grouped so the same sentence
// covers every vector that gap accounts for, rather than one unique
// sentence per vector ID.
const (
	skipReasonMU20Outlier          = "MU-20 statistical outlier detection is not implemented"
	skipReasonMU21Promotion        = "MU-21 statistical promotion-threshold checks are not implemented"
	skipReasonMU22DuplicateEntity  = "MU-22 duplicate/near-duplicate entity detection is not implemented"
	skipReasonSPECPGNotImplemented = "SPEC-PG gate evaluation is not implemented"
)

// skippedConformanceVectors is the one hand-written map this gate holds:
// every vector ID known not to have a test, with the reason. It is checked
// against vectors.Load()'s two matrices on every run -- a stale entry (the
// vector now has a test, or the ID does not exist) fails the build just as
// loudly as a vector with neither a test nor a skip.
//
// Entries are built from an int list plus a shared reason, grouped by
// rule, because writing out 97 individual map-literal entries would bury
// which capability gap each one traces to behind repetition. This is a
// readability choice, not a way to hide these keys from collectVectorIDs
// above: collectVectorIDs only ever looks at a vectorID-keyed struct
// field, so an ordinary map key -- built via fmt.Sprintf here or written
// out literally -- was never something it could see.
func skippedConformanceVectors() map[string]string {
	skips := map[string]string{}
	add := func(reason string, ids ...int) {
		for _, n := range ids {
			skips[fmt.Sprintf("MU-V%d", n)] = reason
		}
	}

	add(skipReasonMU20Outlier, 38, 39, 86, 87)
	add(skipReasonMU21Promotion, 40, 41, 111, 112)
	add(skipReasonMU22DuplicateEntity, 88, 89, 90)

	for n := 1; n <= 53; n++ {
		skips[fmt.Sprintf("PG-V%d", n)] = skipReasonSPECPGNotImplemented
	}

	return skips
}

// TestCollectVectorIDs_OnlyFuncNamesAndVectorIDFields proves collectVectorIDs'
// two-shapes-only rule directly, against small parsed source strings built
// here rather than against decoy code added to a real _test.go file: a
// function name and a vectorID-keyed field both register; a plain string
// literal (the kind a t.Log call or any other incidental string would use)
// and a map literal's string key do not.
func TestCollectVectorIDs_OnlyFuncNamesAndVectorIDFields(t *testing.T) {
	const src = `package fake

import "testing"

func TestCheckMU01_MU_V1(t *testing.T) {}

var cases = []struct {
	vectorID string
	name     string
}{
	{vectorID: "MU-V2", name: "example"},
}

func TestLogsAVectorLikeString(t *testing.T) {
	t.Log("MU-V70")
}

var skips = map[string]string{
	"MU-V69": "reason",
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fake_test.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	got := collectVectorIDs(file)

	for _, want := range []string{"MU-V1", "MU-V2"} {
		if !got[want] {
			t.Errorf("collectVectorIDs missing %s (from a function name or a vectorID field)", want)
		}
	}
	if got["MU-V70"] {
		t.Error("collectVectorIDs collected MU-V70 from a t.Log string literal -- only function names and vectorID fields should count")
	}
	if got["MU-V69"] {
		t.Error("collectVectorIDs collected MU-V69 from a map literal's string key -- only function names and vectorID fields should count")
	}
	if len(got) != 2 {
		t.Errorf("collectVectorIDs returned %d ID(s) %v, want exactly 2", len(got), got)
	}
}
