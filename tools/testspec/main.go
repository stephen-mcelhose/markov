// Command testspec reads "go test -v" output from stdin and re-renders it as
// an RFC 2119 specification. Only sub-tests whose names contain at least one
// RFC 2119 keyword (MUST, MUST NOT, SHOULD, SHOULD NOT, SHALL, SHALL NOT,
// REQUIRED, RECOMMENDED, OPTIONAL, MAY) are shown by default. Pass --all to
// include non-specable sub-tests (useful for debugging).
//
// Usage:
//
//	go test -v ./... | go run ./tools/testspec
//	go test -v ./... | go run ./tools/testspec --all
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	showAll = flag.Bool("all", false, "also show non-specable sub-tests (those without RFC 2119 keywords)")

	// rfc2119 matches any RFC 2119 keyword appearing as a whole word.
	// Order matters: longer phrases must come before their prefixes.
	rfc2119 = regexp.MustCompile(`\b(MUST NOT|MUST|SHOULD NOT|SHOULD|SHALL NOT|SHALL|REQUIRED|RECOMMENDED|OPTIONAL|MAY)\b`)
)

// subResult holds the parsed result of one sub-test run.
type subResult struct {
	testFunc string // e.g. "TestNewKernel"
	subName  string // humanized sub-test name (underscores → spaces)
	passed   bool
}

// pkgResult accumulates everything emitted for one package.
type pkgResult struct {
	pkg        string
	subs       []subResult
	funcOrder  []string          // insertion-ordered test function names
	byFunc     map[string][]subResult
	passed     bool
	buildFailed bool
}

func newPkgResult() pkgResult {
	return pkgResult{byFunc: make(map[string][]subResult)}
}

func isSpecable(name string) bool { return rfc2119.MatchString(name) }

func humanize(s string) string { return strings.ReplaceAll(s, "_", " ") }

// stripTiming removes the trailing " (0.00s)" appended by go test.
func stripTiming(s string) string {
	if i := strings.LastIndex(s, " ("); i >= 0 {
		return s[:i]
	}
	return s
}

// parseResultLine parses "--- PASS: name (xs)" or "--- FAIL: name (xs)".
// Returns the full test name, whether it passed, and whether parsing succeeded.
func parseResultLine(line string) (name string, passed bool, ok bool) {
	switch {
	case strings.HasPrefix(line, "--- PASS: "):
		return stripTiming(strings.TrimPrefix(line, "--- PASS: ")), true, true
	case strings.HasPrefix(line, "--- FAIL: "):
		return stripTiming(strings.TrimPrefix(line, "--- FAIL: ")), false, true
	}
	return "", false, false
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: go test -v ./... | testspec [--all]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var (
		pkgs    []pkgResult
		current = newPkgResult()
	)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4<<20), 4<<20)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "--- PASS: ") || strings.HasPrefix(line, "--- FAIL: "):
			name, passed, ok := parseResultLine(line)
			if !ok {
				continue
			}
			// Only sub-tests (names containing "/") belong to our spec output.
			slashIdx := strings.Index(name, "/")
			if slashIdx < 0 {
				continue
			}
			testFunc := name[:slashIdx]
			subRaw := name[slashIdx+1:]
			sub := subResult{
				testFunc: testFunc,
				subName:  humanize(subRaw),
				passed:   passed,
			}
			if _, seen := current.byFunc[testFunc]; !seen {
				current.funcOrder = append(current.funcOrder, testFunc)
			}
			current.byFunc[testFunc] = append(current.byFunc[testFunc], sub)

		case strings.HasPrefix(line, "ok  \t") || strings.HasPrefix(line, "ok \t"):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				current.pkg = fields[1]
				current.passed = true
				pkgs = append(pkgs, current)
				current = newPkgResult()
			}

		case strings.HasPrefix(line, "FAIL\t") || strings.HasPrefix(line, "FAIL "):
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				current.pkg = fields[1]
				current.buildFailed = strings.Contains(line, "[build failed]")
				pkgs = append(pkgs, current)
				current = newPkgResult()
			}

		case strings.HasPrefix(line, "?   \t"):
			// "?   	pkg [no test files]" — nothing to do.
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "testspec: read error: %v\n", err)
		os.Exit(2)
	}

	// ── Render ───────────────────────────────────────────────────────────────

	var totalSpecable, totalFailed, totalPackages int

	for _, pkg := range pkgs {
		totalPackages++
		fmt.Printf("\n▶ %s\n", pkg.pkg)

		if pkg.buildFailed {
			fmt.Printf("  [BUILD FAILED]\n")
			totalFailed++ // count the package as a failure
			continue
		}

		for _, fn := range pkg.funcOrder {
			subs := pkg.byFunc[fn]

			// collect what to show
			var toShow []subResult
			for _, s := range subs {
				if isSpecable(s.subName) || *showAll {
					toShow = append(toShow, s)
				}
			}
			if len(toShow) == 0 {
				continue
			}

			// strip "Test" prefix from function name for readability
			fnHuman := strings.TrimPrefix(fn, "Test")
			fmt.Printf("\n  %s\n", fnHuman)

			for _, s := range toShow {
				icon := "✓"
				if !s.passed {
					icon = "✗"
					totalFailed++
				}
				if isSpecable(s.subName) {
					totalSpecable++
				}
				fmt.Printf("    %s  %s\n", icon, s.subName)
			}
		}

		if !pkg.passed {
			fmt.Printf("\n  [PACKAGE FAILED]\n")
		}
	}

	// ── Summary ───────────────────────────────────────────────────────────────
	fmt.Println()
	if totalFailed > 0 {
		fmt.Printf("FAIL  %d package(s)  %d specable  %d failed\n", totalPackages, totalSpecable, totalFailed)
		os.Exit(1)
	}
	fmt.Printf("PASS  %d package(s)  %d specable\n", totalPackages, totalSpecable)
}
