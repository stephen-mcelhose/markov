package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceGraph(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	archBytes, err := os.ReadFile(filepath.Join(repoRoot, "docs", "architecture.md"))
	if err != nil {
		t.Fatalf("read architecture.md: %v", err)
	}
	ctxBytes, err := os.ReadFile(filepath.Join(repoRoot, "docs", "surface", "context.jsonld"))
	if err != nil {
		t.Fatalf("read context.jsonld: %v", err)
	}
	shaclBytes, err := os.ReadFile(filepath.Join(repoRoot, "docs", "surface", "surface.shacl.ttl"))
	if err != nil {
		t.Fatalf("read surface.shacl.ttl: %v", err)
	}

	validGraph := func() *SurfaceGraph {
		g, err := ParseArchitectureMarkdown(string(archBytes))
		if err != nil {
			t.Fatalf("ParseArchitectureMarkdown: %v", err)
		}
		return g
	}

	type testCase struct {
		name      string
		mutate    func(g *SurfaceGraph)
		wantValid bool
		wantErr   string
	}

	cases := []testCase{
		{
			name:      "MUST validate the complete canonical surface graph with zero SHACL violations",
			mutate:    func(g *SurfaceGraph) {},
			wantValid: true,
		},
		{
			name: "MUST reject SurfaceGraph missing a required package",
			mutate: func(g *SurfaceGraph) {
				g.Control = nil
			},
			wantValid: false,
			wantErr:   "SurfaceGraphShape",
		},
		{
			name: "MUST reject Function missing mathematical def",
			mutate: func(g *SurfaceGraph) {
				g.Layer1.Kernel.Symbols[1].Def = ""
			},
			wantValid: false,
			wantErr:   "FunctionShape",
		},
		{
			name: "MUST reject Function with non-LaTeX def",
			mutate: func(g *SurfaceGraph) {
				g.Layer1.Kernel.Symbols[1].Def = "plain text without math delimiters"
			},
			wantValid: false,
			wantErr:   "FunctionShape",
		},
		{
			name: "MUST reject Symbol with invalid Go signature syntax",
			mutate: func(g *SurfaceGraph) {
				g.Layer1.Kernel.Symbols[1].Signature = "func NewKernel(P *mat.Dense ("
			},
			wantValid: false,
			wantErr:   "invalid Go signature",
		},
		{
			name: "MUST reject Method with unresolved receiverType IRI",
			mutate: func(g *SurfaceGraph) {
				for _, s := range g.Layer1.Kernel.Symbols {
					if s.Type == "Method" {
						s.ReceiverType = "urn:markov:sym:markov.NonExistentType"
						break
					}
				}
			},
			wantValid: false,
			wantErr:   "unresolved IRI reference",
		},
		{
			name: "MUST reject Source with non-HTTP URL",
			mutate: func(g *SurfaceGraph) {
				g.Layer1.Kernel.Symbols[1].Src[0].URL = "ftp://invalid.example.com"
			},
			wantValid: false,
			wantErr:   "SourceShape",
		},
		{
			name: "MUST reject Invariant without LaTeX expression",
			mutate: func(g *SurfaceGraph) {
				g.Layer1.Kernel.Symbols[1].Invariants[0].Expr = "not latex"
			},
			wantValid: false,
			wantErr:   "InvariantShape",
		},
		{
			name: "MUST reject cyclic package dependencies",
			mutate: func(g *SurfaceGraph) {
				g.Layer1.DependsOn = []string{"urn:markov:pkg:reduce"}
			},
			wantValid: false,
			wantErr:   "cycle detected",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := validGraph()
			tc.mutate(g)
			errs := Validate(g, string(ctxBytes), string(shaclBytes))
			if tc.wantValid && len(errs) > 0 {
				t.Fatalf("unexpected validation errors: %v", errs)
			}
			if !tc.wantValid {
				if len(errs) == 0 {
					t.Fatalf("expected validation error containing %q, got none", tc.wantErr)
				}
				found := false
				for _, e := range errs {
					if strings.Contains(e.Error(), tc.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, errs)
				}
			}
		})
	}
}

func TestRunCLI(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	tmpSurface := filepath.Join(t.TempDir(), "surface.jsonld")

	type testCase struct {
		name     string
		opts     options
		wantCode int
		wantOut  string
	}

	cases := []testCase{
		{
			name: "MUST emit and validate surface.jsonld from docs/architecture.md",
			opts: options{
				archPath:    filepath.Join(repoRoot, "docs", "architecture.md"),
				surfacePath: tmpSurface,
				contextPath: filepath.Join(repoRoot, "docs", "surface", "context.jsonld"),
				shaclPath:   filepath.Join(repoRoot, "docs", "surface", "surface.shacl.ttl"),
				emit:        true,
				checkSync:   true,
			},
			wantCode: 0,
			wantOut:  "PASS  surface graph valid (5 packages, 22 files, 74 symbols, 80 invariants, 56 sources)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			code := run(tc.opts, &buf)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d\noutput:\n%s", code, tc.wantCode, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantOut) {
				t.Fatalf("output = %q, want substring %q", buf.String(), tc.wantOut)
			}
		})
	}
}

func TestVerifyAST(t *testing.T) {
	g := &SurfaceGraph{
		Layer1: &PackageNode{
			ID:      "urn:markov:pkg:markov",
			Type:    "Package",
			Package: "markov",
			Dir:     ".",
			Doc:     &FileNode{ID: "urn:markov:file:markov/doc.go", Type: "File", Path: "doc.go"},
			Errors: &FileNode{
				ID:   "urn:markov:file:markov/errors.go",
				Type: "File",
				Path: "errors.go",
				Symbols: []*SymbolNode{
					{ID: "urn:markov:sym:markov.ErrFoo", Type: "Error", Name: "ErrFoo", Signature: "var ErrFoo error"},
				},
			},
		},
	}

	type testCase struct {
		name      string
		files     map[string]string
		wantValid bool
		wantErr   string
	}

	cases := []testCase{
		{
			name: "MUST succeed when package matches surface signatures and AGENTS.md policies",
			files: map[string]string{
				"doc.go":         "// Package markov doc.\npackage markov\n",
				"errors.go":      "package markov\n\nimport \"errors\"\n\n// ErrFoo is an error.\nvar ErrFoo = errors.New(\"foo\")\n",
				"errors_test.go": "package markov\n\nimport \"testing\"\n\nfunc TestErrFoo(t *testing.T) {}\n",
			},
			wantValid: true,
		},
		{
			name: "MUST reject exported symbol missing Go doc comment",
			files: map[string]string{
				"doc.go":         "// Package markov doc.\npackage markov\n",
				"errors.go":      "package markov\n\nimport \"errors\"\n\nvar ErrFoo = errors.New(\"foo\")\n",
				"errors_test.go": "package markov\n\nimport \"testing\"\n\nfunc TestErrFoo(t *testing.T) {}\n",
			},
			wantValid: false,
			wantErr:   "missing a Go doc comment",
		},
		{
			name: "MUST reject test function named with RFC 2119 keyword",
			files: map[string]string{
				"doc.go":         "// Package markov doc.\npackage markov\n",
				"errors.go":      "package markov\n\nimport \"errors\"\n\n// ErrFoo is an error.\nvar ErrFoo = errors.New(\"foo\")\n",
				"errors_test.go": "package markov\n\nimport \"testing\"\n\nfunc TestMUSTFail(t *testing.T) {}\n",
			},
			wantValid: false,
			wantErr:   "MUST NOT use RFC 2119 keywords in function name",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, errs := VerifyAST(g, dir, []string{"markov"})
			if tc.wantValid && len(errs) > 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if !tc.wantValid {
				found := false
				for _, e := range errs {
					if strings.Contains(e.Error(), tc.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, errs)
				}
			}
		})
	}
}

func TestValidatePersonas(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	personaSHACL := filepath.Join(repoRoot, "docs", "personas", "persona.shacl.ttl")
	realPersonasDir := filepath.Join(repoRoot, "docs", "personas")

	type testCase struct {
		name      string
		dirSetup  func(t *testing.T) string
		wantValid bool
		wantErr   string
	}

	cases := []testCase{
		{
			name: "MUST validate docs/personas/dreyfus-stage-2.md against persona.shacl.ttl",
			dirSetup: func(t *testing.T) string {
				return realPersonasDir
			},
			wantValid: true,
		},
		{
			name: "MUST reject persona document with wrong type in frontmatter",
			dirSetup: func(t *testing.T) string {
				d := t.TempDir()
				bad := "---\ntype: concept\ntitle: Bad Persona\ndescription: A persona with wrong type field.\nstage: Dreyfus Stage 2\ntimestamp: 2026-11-14T19:00:00Z\n---\n## 1. Skill Stage\n## 2. Prior Knowledge\n## 3. Mental Model & Vocabulary Bridge\n## 4. Situational Maxims\n## 5. Tone, Do's & Don'ts\n## 6. Examples Roadmap\n## Sources\n"
				_ = os.WriteFile(filepath.Join(d, "bad.md"), []byte(bad), 0o644)
				return d
			},
			wantValid: false,
			wantErr:   "MUST declare 'type: persona'",
		},
		{
			name: "MUST reject persona document missing required sections",
			dirSetup: func(t *testing.T) string {
				d := t.TempDir()
				incomplete := "---\ntype: persona\ntitle: Incomplete Persona\ndescription: A persona missing required sections.\nstage: Dreyfus Stage 2\ntimestamp: 2026-11-14T19:00:00Z\n---\n## 1. Skill Stage\n"
				_ = os.WriteFile(filepath.Join(d, "incomplete.md"), []byte(incomplete), 0o644)
				return d
			},
			wantValid: false,
			wantErr:   "MUST contain all 7 required sections",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.dirSetup(t)
			_, errs := ValidatePersonas(dir, personaSHACL)
			if tc.wantValid && len(errs) > 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if !tc.wantValid {
				found := false
				for _, e := range errs {
					if strings.Contains(e.Error(), tc.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, errs)
				}
			}
		})
	}
}

func TestRenderDocsAndInspect(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	surfacePath := filepath.Join(repoRoot, "docs", "surface", "surface.jsonld")
	tmpArch := filepath.Join(t.TempDir(), "architecture.md")

	type testCase struct {
		name     string
		opts     options
		wantCode int
		wantOut  string
	}

	cases := []testCase{
		{
			name: "MUST render architecture.md from surface.jsonld and pass round-trip sync check",
			opts: options{
				archPath:    tmpArch,
				surfacePath: surfacePath,
				contextPath: filepath.Join(repoRoot, "docs", "surface", "context.jsonld"),
				shaclPath:   filepath.Join(repoRoot, "docs", "surface", "surface.shacl.ttl"),
				repoRoot:    repoRoot,
				renderDocs:  true,
				checkSync:   true,
			},
			wantCode: 0,
			wantOut:  "PASS  rendered",
		},
		{
			name: "MUST print mathematical specification card when inspecting a matching symbol",
			opts: options{
				surfacePath: surfacePath,
				inspect:     "MFPTDelta",
			},
			wantCode: 0,
			wantOut:  "urn:markov:sym:solve.MFPTDelta",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			code := run(tc.opts, &buf)
			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d\noutput:\n%s", code, tc.wantCode, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantOut) {
				t.Fatalf("output missing %q\nfull output:\n%s", tc.wantOut, buf.String())
			}
		})
	}
}

func TestCobraSubcommands(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	surfacePath := filepath.Join(repoRoot, "docs", "surface", "surface.jsonld")
	archPath := filepath.Join(repoRoot, "docs", "architecture.md")
	ctxPath := filepath.Join(repoRoot, "docs", "surface", "context.jsonld")
	shaclPath := filepath.Join(repoRoot, "docs", "surface", "surface.shacl.ttl")

	type testCase struct {
		name    string
		args    []string
		wantOut string
		wantErr bool
	}

	cases := []testCase{
		{
			name: "MUST execute inspect subcommand and print reverse-graph laws",
			args: []string{
				"inspect", "reduce.Lump",
				"--surface", surfacePath,
			},
			wantOut: "Cross-Symbol Laws Referencing Lump",
		},
		{
			name: "MUST execute validate subcommand and pass all SHACL checks",
			args: []string{
				"validate",
				"--arch", archPath,
				"--surface", surfacePath,
				"--context", ctxPath,
				"--shacl", shaclPath,
				"--root", repoRoot,
			},
			wantOut: "PASS  surface graph valid",
		},
		{
			name: "MUST execute ast subcommand for specified package",
			args: []string{
				"ast", "solve",
				"--arch", archPath,
				"--surface", surfacePath,
				"--context", ctxPath,
				"--shacl", shaclPath,
				"--root", repoRoot,
			},
			wantOut: "PASS  AST & policy verified across 1 implemented package(s)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := NewRootCommand(&buf)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput:\n%s", err, buf.String())
			}
			if !strings.Contains(buf.String(), tc.wantOut) {
				t.Fatalf("output missing %q\nfull output:\n%s", tc.wantOut, buf.String())
			}
		})
	}
}

func TestValidateRunbooks(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	runbookSHACL := filepath.Join(repoRoot, "docs", "runbooks", "runbook.shacl.ttl")
	realRunbooksDir := filepath.Join(repoRoot, "docs", "runbooks")

	type testCase struct {
		name      string
		dirSetup  func(t *testing.T) string
		wantValid bool
		wantErr   string
	}

	cases := []testCase{
		{
			name: "MUST validate docs/runbooks/surface-workflow.md against runbook.shacl.ttl",
			dirSetup: func(t *testing.T) string {
				return realRunbooksDir
			},
			wantValid: true,
		},
		{
			name: "MUST reject runbook missing Preconditions or Verification sections",
			dirSetup: func(t *testing.T) string {
				d := t.TempDir()
				bad := "---\ntype: runbook\ntitle: Incomplete Runbook\ndescription: A runbook missing required sections.\ntimestamp: 2026-11-14T19:00:00Z\n---\n## 1. When to Use This Runbook\n"
				_ = os.WriteFile(filepath.Join(d, "bad.md"), []byte(bad), 0o644)
				return d
			},
			wantValid: false,
			wantErr:   "MUST contain all 6 required sections",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.dirSetup(t)
			_, errs := ValidateRunbooks(dir, runbookSHACL)
			if tc.wantValid && len(errs) > 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if !tc.wantValid {
				found := false
				for _, e := range errs {
					if strings.Contains(e.Error(), tc.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, errs)
				}
			}
		})
	}
}
