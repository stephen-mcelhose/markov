package main

import (
	"strings"
	"testing"
)

func TestIsSpecable(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected bool
	}
	cases := []testCase{
		{name: "MUST is specable", input: "MUST return error when nil", expected: true},
		{name: "MUST NOT is specable", input: "MUST NOT panic on invalid input", expected: true},
		{name: "SHOULD is specable", input: "SHOULD clamp negative entries", expected: true},
		{name: "SHOULD NOT is specable", input: "SHOULD NOT return error on valid input", expected: true},
		{name: "SHALL is specable", input: "SHALL preserve row sums", expected: true},
		{name: "SHALL NOT is specable", input: "SHALL NOT modify the receiver", expected: true},
		{name: "REQUIRED is specable", input: "REQUIRED field is non-nil", expected: true},
		{name: "RECOMMENDED is specable", input: "RECOMMENDED to use tol 1e-9", expected: true},
		{name: "OPTIONAL is specable", input: "OPTIONAL names may be nil", expected: true},
		{name: "MAY is specable", input: "MAY return nil on empty input", expected: true},
		{name: "lowercase must is not specable", input: "must return error", expected: false},
		{name: "mixed case Must is not specable", input: "Must return error", expected: false},
		{name: "example prefix is not specable", input: "example valid kernel", expected: false},
		{name: "empty string is not specable", input: "", expected: false},
		{name: "MUST NOT takes precedence over MUST alone", input: "MUST NOT panic", expected: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsSpecable(tc.input)
			if got != tc.expected {
				t.Errorf("IsSpecable(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

// input is a helper that trims leading whitespace from each line of a
// heredoc-style string, so test inputs can be indented for readability.
func input(s string) string { return strings.TrimLeft(s, "\n") }

func TestRun(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		showAll  bool
		wantCode int
		wantOut  []string // substrings that MUST appear in output
		wantAbsent []string // substrings that MUST NOT appear in output
	}
	cases := []testCase{
		{
			name: "MUST passing sub-test is shown with check mark",
			input: input(`
=== RUN   TestNewKernel
=== RUN   TestNewKernel/MUST_return_error_when_nil
--- PASS: TestNewKernel/MUST_return_error_when_nil (0.00s)
--- PASS: TestNewKernel (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode: 0,
			wantOut:  []string{"✓  MUST return error when nil", "NewKernel", "PASS"},
		},
		{
			name: "MUST failing sub-test is shown with cross",
			input: input(`
=== RUN   TestNewKernel
=== RUN   TestNewKernel/MUST_return_error_when_nil
--- FAIL: TestNewKernel/MUST_return_error_when_nil (0.00s)
--- FAIL: TestNewKernel (0.00s)
FAIL	github.com/x/y	0.001s
`),
			wantCode: 1,
			wantOut:  []string{"✗  MUST return error when nil", "FAIL"},
		},
		{
			name: "non-specable sub-test is hidden by default",
			input: input(`
=== RUN   TestNewKernel
=== RUN   TestNewKernel/example_valid_kernel
--- PASS: TestNewKernel/example_valid_kernel (0.00s)
--- PASS: TestNewKernel (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode:   0,
			wantAbsent: []string{"example valid kernel"},
		},
		{
			name: "non-specable sub-test is shown with showAll",
			input: input(`
=== RUN   TestNewKernel
=== RUN   TestNewKernel/example_valid_kernel
--- PASS: TestNewKernel/example_valid_kernel (0.00s)
--- PASS: TestNewKernel (0.00s)
ok  	github.com/x/y	0.001s
`),
			showAll:  true,
			wantCode: 0,
			wantOut:  []string{"example valid kernel"},
		},
		{
			name: "Test prefix is stripped from function name",
			input: input(`
=== RUN   TestNormalizeRows
=== RUN   TestNormalizeRows/MUST_error_on_zero_row_sum
--- PASS: TestNormalizeRows/MUST_error_on_zero_row_sum (0.00s)
--- PASS: TestNormalizeRows (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode: 0,
			wantOut:  []string{"NormalizeRows"},
			wantAbsent: []string{"TestNormalizeRows"},
		},
		{
			name: "underscores in sub-test name are replaced with spaces",
			input: input(`
=== RUN   TestFoo
=== RUN   TestFoo/MUST_preserve_row_sum_invariant
--- PASS: TestFoo/MUST_preserve_row_sum_invariant (0.00s)
--- PASS: TestFoo (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode: 0,
			wantOut:  []string{"MUST preserve row sum invariant"},
			wantAbsent: []string{"MUST_preserve_row_sum_invariant"},
		},
		{
			name: "build failed package is flagged and counted as failure",
			input: input(`
FAIL	github.com/x/y	[build failed]
`),
			wantCode: 1,
			wantOut:  []string{"[BUILD FAILED]", "FAIL"},
		},
		{
			name: "no test files line is silently ignored",
			input: input(`
?   	github.com/x/y	[no test files]
`),
			wantCode: 0,
			wantOut:  []string{"PASS  0 package(s)"},
		},
		{
			name: "multiple packages are each shown under their own heading",
			input: input(`
=== RUN   TestFoo
=== RUN   TestFoo/MUST_do_A
--- PASS: TestFoo/MUST_do_A (0.00s)
--- PASS: TestFoo (0.00s)
ok  	github.com/x/pkg1	0.001s
=== RUN   TestBar
=== RUN   TestBar/MUST_do_B
--- PASS: TestBar/MUST_do_B (0.00s)
--- PASS: TestBar (0.00s)
ok  	github.com/x/pkg2	0.001s
`),
			wantCode: 0,
			wantOut:  []string{"github.com/x/pkg1", "MUST do A", "github.com/x/pkg2", "MUST do B"},
		},
		{
			name: "summary counts specable tests correctly",
			input: input(`
=== RUN   TestFoo
=== RUN   TestFoo/MUST_do_A
--- PASS: TestFoo/MUST_do_A (0.00s)
=== RUN   TestFoo/MUST_do_B
--- PASS: TestFoo/MUST_do_B (0.00s)
=== RUN   TestFoo/example_ignore_me
--- PASS: TestFoo/example_ignore_me (0.00s)
--- PASS: TestFoo (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode: 0,
			wantOut:  []string{"2 specable"},
		},
		{
			name: "top-level test result without slash is not shown",
			input: input(`
=== RUN   TestFoo
--- PASS: TestFoo (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode: 0,
			// no function block should appear since there are no sub-tests
			wantAbsent: []string{"Foo\n"},
		},
		{
			name: "SHOULD NOT is detected as specable",
			input: input(`
=== RUN   TestFoo
=== RUN   TestFoo/SHOULD_NOT_panic_on_empty_partition
--- PASS: TestFoo/SHOULD_NOT_panic_on_empty_partition (0.00s)
--- PASS: TestFoo (0.00s)
ok  	github.com/x/y	0.001s
`),
			wantCode: 0,
			wantOut:  []string{"SHOULD NOT panic on empty partition"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			code := run(strings.NewReader(tc.input), &buf, tc.showAll)
			out := buf.String()

			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d\noutput:\n%s", code, tc.wantCode, out)
			}
			for _, want := range tc.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\nfull output:\n%s", want, out)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(out, absent) {
					t.Errorf("output contains unexpected %q\nfull output:\n%s", absent, out)
				}
			}
		})
	}
}
