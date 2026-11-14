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
	"io"
	"os"
	"regexp"
	"strings"
)

// rfc2119 matches any RFC 2119 keyword appearing as a whole word.
// Longer phrases must precede their prefixes (MUST NOT before MUST).
var rfc2119 = regexp.MustCompile(`\b(MUST NOT|MUST|SHOULD NOT|SHOULD|SHALL NOT|SHALL|REQUIRED|RECOMMENDED|OPTIONAL|MAY)\b`)

// IsSpecable reports whether name contains at least one RFC 2119 keyword.
func IsSpecable(name string) bool { return rfc2119.MatchString(name) }

func humanize(s string) string { return strings.ReplaceAll(s, "_", " ") }

func stripTiming(s string) string {
	if i := strings.LastIndex(s, " ("); i >= 0 {
		return s[:i]
	}
	return s
}

func parseResultLine(line string) (name string, passed bool, ok bool) {
	switch {
	case strings.HasPrefix(line, "--- PASS: "):
		return stripTiming(strings.TrimPrefix(line, "--- PASS: ")), true, true
	case strings.HasPrefix(line, "--- FAIL: "):
		return stripTiming(strings.TrimPrefix(line, "--- FAIL: ")), false, true
	}
	return "", false, false
}

type subResult struct {
	testFunc string
	subName  string // humanized
	passed   bool
}

type pkgResult struct {
	pkg         string
	funcOrder   []string
	byFunc      map[string][]subResult
	passed      bool
	buildFailed bool
}

func newPkgResult() pkgResult {
	return pkgResult{byFunc: make(map[string][]subResult)}
}

// run parses go test -v output from r and writes a spec report to w.
// Returns 0 on success, 1 if any test failed or a package failed to build.
func run(r io.Reader, w io.Writer, showAll bool) int {
	var pkgs []pkgResult
	current := newPkgResult()

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4<<20), 4<<20)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "--- PASS: ") || strings.HasPrefix(line, "--- FAIL: "):
			name, passed, ok := parseResultLine(line)
			if !ok {
				continue
			}
			slashIdx := strings.Index(name, "/")
			if slashIdx < 0 {
				continue // top-level result; not a sub-test
			}
			testFunc := name[:slashIdx]
			sub := subResult{
				testFunc: testFunc,
				subName:  humanize(name[slashIdx+1:]),
				passed:   passed,
			}
			if _, seen := current.byFunc[testFunc]; !seen {
				current.funcOrder = append(current.funcOrder, testFunc)
			}
			current.byFunc[testFunc] = append(current.byFunc[testFunc], sub)

		case strings.HasPrefix(line, "ok  \t") || strings.HasPrefix(line, "ok \t"):
			if fields := strings.Fields(line); len(fields) >= 2 {
				current.pkg = fields[1]
				current.passed = true
				pkgs = append(pkgs, current)
				current = newPkgResult()
			}

		case strings.HasPrefix(line, "FAIL\t") || strings.HasPrefix(line, "FAIL "):
			if fields := strings.Fields(line); len(fields) >= 2 {
				current.pkg = fields[1]
				current.buildFailed = strings.Contains(line, "[build failed]")
				pkgs = append(pkgs, current)
				current = newPkgResult()
			}
		}
	}

	var totalSpecable, totalFailed, totalPackages int

	for _, pkg := range pkgs {
		totalPackages++
		fmt.Fprintf(w, "\n▶ %s\n", pkg.pkg)

		if pkg.buildFailed {
			fmt.Fprintf(w, "  [BUILD FAILED]\n")
			totalFailed++
			continue
		}

		for _, fn := range pkg.funcOrder {
			var toShow []subResult
			for _, s := range pkg.byFunc[fn] {
				if IsSpecable(s.subName) || showAll {
					toShow = append(toShow, s)
				}
			}
			if len(toShow) == 0 {
				continue
			}
			fmt.Fprintf(w, "\n  %s\n", strings.TrimPrefix(fn, "Test"))
			for _, s := range toShow {
				icon := "✓"
				if !s.passed {
					icon = "✗"
					totalFailed++
				}
				if IsSpecable(s.subName) {
					totalSpecable++
				}
				fmt.Fprintf(w, "    %s  %s\n", icon, s.subName)
			}
		}

		if !pkg.passed && !pkg.buildFailed {
			fmt.Fprintf(w, "\n  [PACKAGE FAILED]\n")
		}
	}

	fmt.Fprintln(w)
	if totalFailed > 0 {
		fmt.Fprintf(w, "FAIL  %d package(s)  %d specable  %d failed\n", totalPackages, totalSpecable, totalFailed)
		return 1
	}
	fmt.Fprintf(w, "PASS  %d package(s)  %d specable\n", totalPackages, totalSpecable)
	return 0
}

func main() {
	showAll := flag.Bool("all", false, "also show non-specable sub-tests")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: go test -v ./... | testspec [--all]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	os.Exit(run(os.Stdin, os.Stdout, *showAll))
}
