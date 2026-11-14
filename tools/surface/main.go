// Command surface validates the markov API surface graph (docs/surface/surface.jsonld)
// against its JSON-LD context (docs/surface/context.jsonld), its SHACL shapes
// (docs/surface/surface.shacl.ttl), Go signature syntax, and docs/architecture.md.
//
// Usage:
//
//	go run ./tools/surface validate
//	go run ./tools/surface ast [pkg...]
//	go run ./tools/surface inspect <pattern>
//	go run ./tools/surface render
//	go run ./tools/surface emit
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// PredicateSchema documents the core symbol predicates in the surface graph.
type PredicateSchema struct {
	Def        string `json:"def"`
	Req        string `json:"req"`
	Err        string `json:"err"`
	Src        string `json:"src"`
	Invariants string `json:"invariants"`
}

// SurfaceGraph is the root JSON-LD document matching:
// { "@context": "./context.jsonld", "@id": "urn:markov:surface", "@type": "SurfaceGraph", "module": ..., "schema": ..., "layer1": { "kernel": ... }, "reduce": { ... }, ... }
type SurfaceGraph struct {
	Context string           `json:"@context"`
	ID      string           `json:"@id"`
	Type    string           `json:"@type"`
	Module  string           `json:"module"`
	Schema  *PredicateSchema `json:"schema"`
	Layer1  *PackageNode     `json:"layer1"`
	Reduce  *PackageNode     `json:"reduce"`
	Solve   *PackageNode     `json:"solve"`
	Infer   *PackageNode     `json:"infer"`
	Control *PackageNode     `json:"control"`
}

// PackageNode represents a Go package in the surface graph.
type PackageNode struct {
	ID            string   `json:"@id"`
	Type          string   `json:"@type"`
	Package       string   `json:"package"`
	ImportPath    string   `json:"importPath"`
	Dir           string   `json:"dir"`
	FileNames     []string `json:"files"`
	ErrorNames    []string `json:"errorList,omitempty"`
	DependsOn     []string `json:"dependsOn,omitempty"`
	KnownUnknowns []string `json:"known_unknowns,omitempty"`

	// File edges keyed by file stem (matching context.jsonld).
	Doc          *FileNode `json:"doc,omitempty"`
	Errors       *FileNode `json:"errors,omitempty"`
	Kernel       *FileNode `json:"kernel,omitempty"`
	Distribution *FileNode `json:"distribution,omitempty"`
	Partition    *FileNode `json:"partition,omitempty"`
	Trace        *FileNode `json:"trace,omitempty"`
	Lump         *FileNode `json:"lump,omitempty"`
	Bisim        *FileNode `json:"bisim,omitempty"`
	Stationary   *FileNode `json:"stationary,omitempty"`
	Fundamental  *FileNode `json:"fundamental,omitempty"`
	MFPT         *FileNode `json:"mfpt,omitempty"`
	Classes      *FileNode `json:"classes,omitempty"`
	Entropy      *FileNode `json:"entropy,omitempty"`
	Estimate     *FileNode `json:"estimate,omitempty"`
	Order        *FileNode `json:"order,omitempty"`
	MDP          *FileNode `json:"mdp,omitempty"`
	DP           *FileNode `json:"dp,omitempty"`
}

// Files returns all non-nil FileNodes in the package's declared FileNames order.
func (p *PackageNode) Files() []*FileNode {
	if p == nil {
		return nil
	}
	byPath := map[string]*FileNode{}
	candidates := []*FileNode{
		p.Doc, p.Errors, p.Kernel, p.Distribution, p.Partition,
		p.Trace, p.Lump, p.Bisim, p.Stationary, p.Fundamental,
		p.MFPT, p.Classes, p.Entropy, p.Estimate, p.Order,
		p.MDP, p.DP,
	}
	for _, f := range candidates {
		if f != nil {
			byPath[f.Path] = f
		}
	}
	var out []*FileNode
	seen := map[string]bool{}
	for _, name := range p.FileNames {
		if f, ok := byPath[name]; ok {
			out = append(out, f)
			seen[name] = true
		}
	}
	for _, f := range candidates {
		if f != nil && !seen[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

func (p *PackageNode) setFile(stem string, f *FileNode) error {
	switch stem {
	case "doc":
		p.Doc = f
	case "errors":
		p.Errors = f
	case "kernel":
		p.Kernel = f
	case "distribution":
		p.Distribution = f
	case "partition":
		p.Partition = f
	case "trace":
		p.Trace = f
	case "lump":
		p.Lump = f
	case "bisim":
		p.Bisim = f
	case "stationary":
		p.Stationary = f
	case "fundamental":
		p.Fundamental = f
	case "mfpt":
		p.MFPT = f
	case "classes":
		p.Classes = f
	case "entropy":
		p.Entropy = f
	case "estimate":
		p.Estimate = f
	case "order":
		p.Order = f
	case "mdp":
		p.MDP = f
	case "dp":
		p.DP = f
	default:
		return fmt.Errorf("unknown file stem %q", stem)
	}
	return nil
}

// FileNode represents a .go source file within a package.
type FileNode struct {
	ID      string        `json:"@id"`
	Type    string        `json:"@type"`
	Path    string        `json:"path"`
	Symbols []*SymbolNode `json:"symbols,omitempty"`
}

// SymbolNode represents an exported Error, Type, Function, or Method.
type SymbolNode struct {
	ID           string           `json:"@id"`
	Type         string           `json:"@type"`
	Name         string           `json:"name"`
	ReceiverType string           `json:"receiverType,omitempty"`
	Signature    string           `json:"signature"`
	Def          string           `json:"def,omitempty"`
	Req          string           `json:"req,omitempty"`
	Err          string           `json:"err,omitempty"`
	DelegatesTo  string           `json:"delegatesTo,omitempty"`
	References   []string         `json:"references,omitempty"`
	Src          []*SourceNode    `json:"src,omitempty"`
	Invariants   []*InvariantNode `json:"invariants,omitempty"`
}

// SourceNode represents a primary literature or implementation reference.
type SourceNode struct {
	Type     string `json:"@type"`
	Citation string `json:"citation"`
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
}

// InvariantNode represents an algebraic or probabilistic invariant (⊢).
type InvariantNode struct {
	ID         string   `json:"@id"`
	Type       string   `json:"@type"`
	Name       string   `json:"name"`
	Expr       string   `json:"expr"`
	References []string `json:"references,omitempty"`
}

// ── Architecture Markdown Parser ──────────────────────────────────────────────

var (
	reMethodSig = regexp.MustCompile(`^func\s+\((?:[a-zA-Z0-9_]+\s+)?\*?([A-Z][A-Za-z0-9_]*)\)\s+([A-Z][A-Za-z0-9_]*)\(`)
	reFuncSig   = regexp.MustCompile(`^func\s+([A-Z][A-Za-z0-9_]*)\(`)
	reTypeSig   = regexp.MustCompile(`^type\s+([A-Z][A-Za-z0-9_]*)\s+`)
	reVarSig    = regexp.MustCompile(`^var\s+(Err[A-Z][A-Za-z0-9_]*)\s+error$`)
	reMdLink    = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+"([^"]+)")?\)`)
	reNonAlpha  = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

// ParseArchitectureMarkdown parses docs/architecture.md into a SurfaceGraph.
func ParseArchitectureMarkdown(md string) (*SurfaceGraph, error) {
	g := &SurfaceGraph{
		Context: "./context.jsonld",
		ID:      "urn:markov:surface",
		Type:    "SurfaceGraph",
		Module:  "github.com/stephen-mcelhose/markov",
		Schema: &PredicateSchema{
			Def:        "Mathematical definition",
			Req:        "Domain preconditions",
			Err:        "Boundary/error behavior",
			Src:        "Primary literature or reference",
			Invariants: "Algebraic/probabilistic invariants (tol-bounded)",
		},
	}

	blocks := extractTextBlocks(md)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no ```text blocks found in architecture markdown")
	}

	for _, block := range blocks {
		pkg, err := parsePackageBlock(block)
		if err != nil {
			return nil, err
		}
		switch pkg.Package {
		case "markov":
			g.Layer1 = pkg
		case "reduce":
			g.Reduce = pkg
		case "solve":
			g.Solve = pkg
		case "infer":
			g.Infer = pkg
		case "control":
			g.Control = pkg
		default:
			return nil, fmt.Errorf("unexpected package %q", pkg.Package)
		}
	}

	resolveCrossReferences(g)
	return g, nil
}

func extractTextBlocks(md string) []string {
	var blocks []string
	lines := strings.Split(md, "\n")
	inBlock := false
	var cur []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock && strings.HasPrefix(trimmed, "```text") {
			inBlock = true
			cur = nil
			continue
		}
		if inBlock && trimmed == "```" {
			inBlock = false
			blocks = append(blocks, strings.Join(cur, "\n"))
			continue
		}
		if inBlock {
			cur = append(cur, line)
		}
	}
	return blocks
}

func parsePackageBlock(block string) (*PackageNode, error) {
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty block")
	}
	rootHeader := strings.TrimSpace(lines[0])
	pkgDir := strings.TrimSuffix(rootHeader, "/")
	pkgName := pkgDir
	importPath := "github.com/stephen-mcelhose/markov/" + pkgName
	dir := pkgName
	var deps []string
	switch pkgName {
	case "markov":
		importPath = "github.com/stephen-mcelhose/markov"
		dir = "."
	case "reduce", "solve", "infer":
		deps = []string{"urn:markov:pkg:markov"}
	case "control":
		deps = []string{"urn:markov:pkg:markov", "urn:markov:pkg:reduce"}
	}

	pkg := &PackageNode{
		ID:         "urn:markov:pkg:" + pkgName,
		Type:       "Package",
		Package:    pkgName,
		ImportPath: importPath,
		Dir:        dir,
		DependsOn:  deps,
	}

	var curFile *FileNode
	var curSym *SymbolNode
	inKnownUnknowns := false

	for _, rawLine := range lines[1:] {
		if strings.TrimSpace(rawLine) == "" {
			continue
		}
		level, content, ok := parseTreeLine(rawLine)
		if !ok {
			continue
		}

		switch level {
		case 1:
			inKnownUnknowns = false
			curSym = nil
			curFile = nil
			if content == "known_unknowns.list" {
				inKnownUnknowns = true
				continue
			}
			if !strings.HasSuffix(content, ".go") {
				continue // skip Makefile, README.md, etc.
			}
			stem := strings.TrimSuffix(content, ".go")
			f := &FileNode{
				ID:   fmt.Sprintf("urn:markov:file:%s/%s", pkgName, content),
				Type: "File",
				Path: content,
			}
			if err := pkg.setFile(stem, f); err != nil {
				return nil, err
			}
			pkg.FileNames = append(pkg.FileNames, content)
			curFile = f

		case 2:
			if inKnownUnknowns {
				pkg.KnownUnknowns = append(pkg.KnownUnknowns, content)
				continue
			}
			if curFile == nil {
				continue
			}
			sym, err := parseSymbolHeader(pkgName, content)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", pkgName, curFile.Path, err)
			}
			if sym.Type == "Error" {
				pkg.ErrorNames = append(pkg.ErrorNames, sym.Name)
			}
			curFile.Symbols = append(curFile.Symbols, sym)
			curSym = sym

		case 3:
			if curSym == nil {
				continue
			}
			if err := attachPredicate(pkgName, curSym, content); err != nil {
				return nil, fmt.Errorf("%s (%s): %w", curSym.ID, curFile.Path, err)
			}
		}
	}

	return pkg, nil
}

// parseTreeLine returns the nesting depth (1 = file, 2 = symbol, 3 = predicate)
// and trimmed payload of a box-drawing tree line.
func parseTreeLine(line string) (int, string, bool) {
	idx := strings.Index(line, "├── ")
	if idx < 0 {
		idx = strings.Index(line, "└── ")
	}
	if idx < 0 {
		return 0, "", false
	}
	prefix := line[:idx]
	// Count number of 4-column indents before the branch marker.
	// In UTF-8 box drawing, "│   " is 6 bytes (3-byte '│' + 3 spaces) or "    " is 4 bytes.
	runes := []rune(prefix)
	level := (len(runes) / 4) + 1
	content := strings.TrimSpace(line[idx+len("├── "):])
	return level, content, true
}

func parseSymbolHeader(pkgName, sig string) (*SymbolNode, error) {
	switch {
	case strings.HasPrefix(sig, "var "):
		m := reVarSig.FindStringSubmatch(sig)
		if m == nil {
			return nil, fmt.Errorf("invalid error var signature: %q", sig)
		}
		name := m[1]
		return &SymbolNode{
			ID:        fmt.Sprintf("urn:markov:sym:%s.%s", pkgName, name),
			Type:      "Error",
			Name:      name,
			Signature: sig,
		}, nil

	case strings.HasPrefix(sig, "type "):
		m := reTypeSig.FindStringSubmatch(sig)
		if m == nil {
			return nil, fmt.Errorf("invalid type signature: %q", sig)
		}
		name := m[1]
		return &SymbolNode{
			ID:        fmt.Sprintf("urn:markov:sym:%s.%s", pkgName, name),
			Type:      "Type",
			Name:      name,
			Signature: sig,
		}, nil

	case strings.HasPrefix(sig, "func ("):
		m := reMethodSig.FindStringSubmatch(sig)
		if m == nil {
			return nil, fmt.Errorf("invalid method signature: %q", sig)
		}
		recvType, methodName := m[1], m[2]
		qualName := recvType + "." + methodName
		return &SymbolNode{
			ID:           fmt.Sprintf("urn:markov:sym:%s.%s", pkgName, qualName),
			Type:         "Method",
			Name:         qualName,
			ReceiverType: fmt.Sprintf("urn:markov:sym:%s.%s", pkgName, recvType),
			Signature:    sig,
		}, nil

	case strings.HasPrefix(sig, "func "):
		m := reFuncSig.FindStringSubmatch(sig)
		if m == nil {
			return nil, fmt.Errorf("invalid func signature: %q", sig)
		}
		name := m[1]
		return &SymbolNode{
			ID:        fmt.Sprintf("urn:markov:sym:%s.%s", pkgName, name),
			Type:      "Function",
			Name:      name,
			Signature: sig,
		}, nil

	default:
		return nil, fmt.Errorf("unrecognized symbol declaration: %q", sig)
	}
}

func attachPredicate(pkgName string, sym *SymbolNode, line string) error {
	switch {
	case strings.HasPrefix(line, "def:"):
		sym.Def = strings.TrimSpace(strings.TrimPrefix(line, "def:"))
	case strings.HasPrefix(line, "req:"):
		sym.Req = strings.TrimSpace(strings.TrimPrefix(line, "req:"))
	case strings.HasPrefix(line, "err:"):
		sym.Err = strings.TrimSpace(strings.TrimPrefix(line, "err:"))
	case strings.HasPrefix(line, "src:"):
		raw := strings.TrimSpace(strings.TrimPrefix(line, "src:"))
		matches := reMdLink.FindAllStringSubmatch(raw, -1)
		if len(matches) == 0 {
			return fmt.Errorf("src line contains no markdown links: %q", raw)
		}
		for _, m := range matches {
			src := &SourceNode{
				Type:     "Source",
				Citation: m[1],
				URL:      m[2],
			}
			if len(m) > 3 && m[3] != "" {
				src.Title = m[3]
			}
			sym.Src = append(sym.Src, src)
		}
	case strings.HasPrefix(line, "⊢"):
		rest := strings.TrimSpace(strings.TrimPrefix(line, "⊢"))
		colonIdx := strings.Index(rest, ":")
		if colonIdx < 0 {
			return fmt.Errorf("invariant missing colon separator: %q", line)
		}
		invName := strings.TrimSpace(rest[:colonIdx])
		invExpr := strings.TrimSpace(rest[colonIdx+1:])
		slug := strings.Trim(reNonAlpha.ReplaceAllString(invName, "_"), "_")
		inv := &InvariantNode{
			ID:   fmt.Sprintf("urn:markov:inv:%s.%s.%s", pkgName, sym.Name, slug),
			Type: "Invariant",
			Name: invName,
			Expr: invExpr,
		}
		sym.Invariants = append(sym.Invariants, inv)
	default:
		return fmt.Errorf("unrecognized predicate line: %q", line)
	}
	return nil
}

// resolveCrossReferences populates delegatesTo and references edges across the graph.
func resolveCrossReferences(g *SurfaceGraph) {
	pkgs := []*PackageNode{g.Layer1, g.Reduce, g.Solve, g.Infer, g.Control}

	// Collect all symbol IDs by package and short name.
	byID := map[string]*SymbolNode{}
	byPkgAndName := map[string]string{}
	var markovTypes []*SymbolNode
	var markovErrors []*SymbolNode

	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files() {
			for _, s := range f.Symbols {
				byID[s.ID] = s
				byPkgAndName[pkg.Package+"."+s.Name] = s.ID
				if pkg.Package == "markov" && s.Type == "Type" {
					markovTypes = append(markovTypes, s)
				}
				if pkg.Package == "markov" && s.Type == "Error" {
					markovErrors = append(markovErrors, s)
				}
			}
		}
	}

	// Delegation pairs: Must* wrappers and top-level/method dual forms.
	delegations := map[string]string{
		"urn:markov:sym:markov.NewKernel":            "urn:markov:sym:markov.NewKernelWithTol",
		"urn:markov:sym:markov.MustNewKernel":        "urn:markov:sym:markov.NewKernel",
		"urn:markov:sym:markov.Kernel.Submatrix":     "urn:markov:sym:markov.Submatrix",
		"urn:markov:sym:markov.NormalizeRows":        "urn:markov:sym:markov.Kernel.NormalizeRows",
		"urn:markov:sym:markov.MustNewDistribution":  "urn:markov:sym:markov.NewDistribution",
		"urn:markov:sym:markov.ValidateDistribution": "urn:markov:sym:markov.Distribution.Validate",
		"urn:markov:sym:markov.RestrictDistribution": "urn:markov:sym:markov.Distribution.Restrict",
		"urn:markov:sym:markov.ProjectDistribution":  "urn:markov:sym:markov.Distribution.Project",
		"urn:markov:sym:markov.ValidatePartition":    "urn:markov:sym:markov.Partition.Validate",
		"urn:markov:sym:markov.IndicatorMatrix":      "urn:markov:sym:markov.Partition.IndicatorMatrix",
		"urn:markov:sym:markov.DistributorMatrix":    "urn:markov:sym:markov.Partition.DistributorMatrix",
		"urn:markov:sym:reduce.MustTrace":            "urn:markov:sym:reduce.Trace",
		"urn:markov:sym:reduce.MustLump":             "urn:markov:sym:reduce.Lump",
		"urn:markov:sym:solve.MustStationary":        "urn:markov:sym:solve.Stationary",
		"urn:markov:sym:control.ValueIteration":      "urn:markov:sym:control.MDP.ValueIteration",
		"urn:markov:sym:control.PolicyIteration":     "urn:markov:sym:control.MDP.PolicyIteration",
		"urn:markov:sym:control.MDPBisimulation":     "urn:markov:sym:control.MDP.CoarsestPartition",
	}

	for from, to := range delegations {
		if s, ok := byID[from]; ok {
			s.DelegatesTo = to
		}
	}

	// Populate symbol references from signatures, err texts, and invariant expressions.
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files() {
			for _, s := range f.Symbols {
				refSet := map[string]bool{}
				addRef := func(targetID string) {
					if targetID != "" && targetID != s.ID {
						refSet[targetID] = true
					}
				}

				if s.DelegatesTo != "" {
					addRef(s.DelegatesTo)
				}

				// Check references to Layer 1 domain types in signature.
				for _, t := range markovTypes {
					if strings.Contains(s.Signature, "markov."+t.Name) ||
						(pkg.Package == "markov" && containsWord(s.Signature, t.Name)) {
						addRef(t.ID)
					}
				}

				// Check references to same-package types in signature.
				for _, f2 := range pkg.Files() {
					for _, s2 := range f2.Symbols {
						if s2.Type == "Type" && containsWord(s.Signature, s2.Name) {
							addRef(s2.ID)
						}
					}
				}

				// Check references to sentinel errors in err specification.
				for _, e := range markovErrors {
					if strings.Contains(s.Err, e.Name) {
						addRef(e.ID)
					}
				}

				if len(refSet) > 0 {
					refs := make([]string, 0, len(refSet))
					for r := range refSet {
						refs = append(refs, r)
					}
					sort.Strings(refs)
					s.References = refs
				}

				// Resolve invariant cross-references.
				for _, inv := range s.Invariants {
					invRefSet := map[string]bool{}
					for key, id := range byPkgAndName {
						parts := strings.SplitN(key, ".", 2)
						short := parts[1]
						if strings.Contains(short, ".") || len(short) < 4 {
							continue
						}
						if strings.Contains(inv.Expr, "\\text{"+short+"}") ||
							strings.Contains(inv.Expr, "."+short+"(") {
							invRefSet[id] = true
						}
					}
					if len(invRefSet) > 0 {
						irefs := make([]string, 0, len(invRefSet))
						for r := range invRefSet {
							irefs = append(irefs, r)
						}
						sort.Strings(irefs)
						inv.References = irefs
					}
				}
			}
		}
	}
}

func containsWord(haystack, word string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString(haystack)
}

// ── SHACL Parser & Validator ──────────────────────────────────────────────────

// SHACLShape represents a parsed sh:NodeShape from surface.shacl.ttl.
type SHACLShape struct {
	Name        string
	TargetClass string
	NodeKind    string
	Pattern     *regexp.Regexp
	Properties  []SHACLProperty
}

// SHACLProperty represents a parsed sh:property blank node constraint.
type SHACLProperty struct {
	Path      string
	Datatype  string
	Class     string
	NodeKind  string
	MinCount  int
	MaxCount  int // -1 means unbounded
	MinLength int
	Pattern   *regexp.Regexp
	Message   string
}

var (
	reShapeHeader = regexp.MustCompile(`^markov:([A-Za-z0-9_]+)\s*$`)
	reTargetClass = regexp.MustCompile(`sh:targetClass\s+markov:([A-Za-z0-9_]+)`)
	reNodeKind    = regexp.MustCompile(`sh:nodeKind\s+sh:([A-Za-z0-9_]+)`)
	reShPath      = regexp.MustCompile(`sh:path\s+markov:([A-Za-z0-9_]+)`)
	reShDatatype  = regexp.MustCompile(`sh:datatype\s+xsd:([A-Za-z0-9_]+)`)
	reShClass     = regexp.MustCompile(`sh:class\s+markov:([A-Za-z0-9_]+)`)
	reShMinCount  = regexp.MustCompile(`sh:minCount\s+(\d+)`)
	reShMaxCount  = regexp.MustCompile(`sh:maxCount\s+(\d+)`)
	reShMinLength = regexp.MustCompile(`sh:minLength\s+(\d+)`)
	reShPattern   = regexp.MustCompile(`sh:pattern\s+"((?:[^"\\]|\\.)*)"`)
	reShMessage   = regexp.MustCompile(`sh:message\s+"((?:[^"\\]|\\.)*)"`)
)

// ParseSHACLShapes parses the Turtle SHACL shapes in surface.shacl.ttl.
func ParseSHACLShapes(ttl string) (map[string]*SHACLShape, error) {
	shapes := map[string]*SHACLShape{}
	lines := strings.Split(ttl, "\n")

	var curShape *SHACLShape
	var curProp *SHACLProperty
	inProp := false

	for lineNum, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "@prefix") {
			continue
		}

		if m := reShapeHeader.FindStringSubmatch(line); m != nil {
			curShape = &SHACLShape{Name: m[1]}
			continue
		}
		if curShape == nil {
			continue
		}

		if strings.HasPrefix(line, "sh:property") && strings.Contains(line, "[") {
			inProp = true
			curProp = &SHACLProperty{MaxCount: -1}
			continue
		}

		if inProp {
			if strings.HasPrefix(line, "]") {
				curShape.Properties = append(curShape.Properties, *curProp)
				curProp = nil
				inProp = false
				if strings.HasSuffix(line, ".") {
					shapes[curShape.TargetClass] = curShape
					curShape = nil
				}
				continue
			}
			if m := reShPath.FindStringSubmatch(line); m != nil {
				curProp.Path = m[1]
			}
			if m := reShDatatype.FindStringSubmatch(line); m != nil {
				curProp.Datatype = m[1]
			}
			if m := reShClass.FindStringSubmatch(line); m != nil {
				curProp.Class = m[1]
			}
			if m := reNodeKind.FindStringSubmatch(line); m != nil {
				curProp.NodeKind = m[1]
			}
			if m := reShMinCount.FindStringSubmatch(line); m != nil {
				curProp.MinCount, _ = strconv.Atoi(m[1])
			}
			if m := reShMaxCount.FindStringSubmatch(line); m != nil {
				curProp.MaxCount, _ = strconv.Atoi(m[1])
			}
			if m := reShMinLength.FindStringSubmatch(line); m != nil {
				curProp.MinLength, _ = strconv.Atoi(m[1])
			}
			if m := reShPattern.FindStringSubmatch(line); m != nil {
				unescaped := strings.ReplaceAll(m[1], `\\`, `\`)
				re, err := regexp.Compile(unescaped)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid sh:pattern %q: %w", lineNum+1, m[1], err)
				}
				curProp.Pattern = re
			}
			if m := reShMessage.FindStringSubmatch(line); m != nil {
				curProp.Message = m[1]
			}
			continue
		}

		// Shape-level constraints.
		if m := reTargetClass.FindStringSubmatch(line); m != nil {
			curShape.TargetClass = m[1]
		}
		if m := reNodeKind.FindStringSubmatch(line); m != nil {
			curShape.NodeKind = m[1]
		}
		if m := reShPattern.FindStringSubmatch(line); m != nil {
			unescaped := strings.ReplaceAll(m[1], `\\`, `\`)
			re, err := regexp.Compile(unescaped)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid shape sh:pattern %q: %w", lineNum+1, m[1], err)
			}
			curShape.Pattern = re
		}
		if strings.HasSuffix(line, ".") {
			shapes[curShape.TargetClass] = curShape
			curShape = nil
		}
	}

	if len(shapes) == 0 {
		return nil, fmt.Errorf("no SHACL NodeShapes parsed from TTL")
	}
	return shapes, nil
}

// Validate checks a SurfaceGraph against the JSON-LD context, SHACL shapes,
// Go AST parser rules for signatures, and referential integrity.
func Validate(g *SurfaceGraph, contextJSON, shaclTTL string) []error {
	var errs []error
	addErr := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	// 1. Validate JSON-LD @context file is valid JSON and defines all required terms.
	var ctxDoc struct {
		Context map[string]any `json:"@context"`
	}
	if err := json.Unmarshal([]byte(contextJSON), &ctxDoc); err != nil {
		addErr("context.jsonld: invalid JSON: %v", err)
		return errs
	}
	requiredTerms := []string{
		"SurfaceGraph", "Package", "File", "Error", "Type", "Function", "Method", "Source", "Invariant",
		"layer1", "reduce", "solve", "infer", "control",
		"package", "importPath", "dir", "dependsOn", "known_unknowns",
		"path", "symbols", "name", "signature", "def", "req", "err",
		"receiverType", "delegatesTo", "references", "src", "invariants",
		"citation", "url", "title", "expr",
	}
	for _, term := range requiredTerms {
		if _, ok := ctxDoc.Context[term]; !ok {
			addErr("context.jsonld: missing required term %q", term)
		}
	}

	// 2. Parse SHACL shapes.
	shapes, err := ParseSHACLShapes(shaclTTL)
	if err != nil {
		addErr("surface.shacl.ttl: %v", err)
		return errs
	}
	requiredClasses := []string{
		"SurfaceGraph", "Package", "File", "Error", "Type", "Function", "Method", "Source", "Invariant",
	}
	for _, cls := range requiredClasses {
		if _, ok := shapes[cls]; !ok {
			addErr("surface.shacl.ttl: missing NodeShape for targetClass markov:%s", cls)
		}
	}
	if len(errs) > 0 {
		return errs
	}

	// 3. Index all IRI nodes in the graph for uniqueness and class lookup.
	nodeClass := map[string]string{}
	recordNode := func(id, typ string) {
		if id == "" {
			addErr("node of type %q is missing @id", typ)
			return
		}
		if prev, exists := nodeClass[id]; exists {
			addErr("duplicate node @id %q (types %s and %s)", id, prev, typ)
			return
		}
		nodeClass[id] = typ
	}

	recordNode(g.ID, g.Type)
	pkgs := []*PackageNode{g.Layer1, g.Reduce, g.Solve, g.Infer, g.Control}
	var nonNilPkgs []*PackageNode
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		nonNilPkgs = append(nonNilPkgs, pkg)
		recordNode(pkg.ID, pkg.Type)
		for _, f := range pkg.Files() {
			recordNode(f.ID, f.Type)
			for _, sym := range f.Symbols {
				recordNode(sym.ID, sym.Type)
				for _, inv := range sym.Invariants {
					recordNode(inv.ID, inv.Type)
				}
			}
		}
	}

	// Helper to validate a node against its SHACL shape.
	checkShape := func(nodeID, nodeType string, props map[string][]string) {
		shape, ok := shapes[nodeType]
		if !ok {
			addErr("node %s has unknown @type %q", nodeID, nodeType)
			return
		}
		if shape.NodeKind == "IRI" && nodeID == "" {
			addErr("[%s] node MUST have an IRI @id", shape.Name)
		}
		if shape.Pattern != nil && !shape.Pattern.MatchString(nodeID) {
			addErr("[%s] node @id %q does not match pattern %s", shape.Name, nodeID, shape.Pattern.String())
		}
		for _, p := range shape.Properties {
			vals := props[p.Path]
			if p.MinCount > 0 && len(vals) < p.MinCount {
				addErr("[%s] %s: %s (path markov:%s got %d values, minCount %d)",
					shape.Name, nodeID, p.Message, p.Path, len(vals), p.MinCount)
				continue
			}
			if p.MaxCount >= 0 && len(vals) > p.MaxCount {
				addErr("[%s] %s: %s (path markov:%s got %d values, maxCount %d)",
					shape.Name, nodeID, p.Message, p.Path, len(vals), p.MaxCount)
				continue
			}
			for _, v := range vals {
				if p.MinLength > 0 && len(v) < p.MinLength {
					addErr("[%s] %s: %s (path markov:%s value %q shorter than minLength %d)",
						shape.Name, nodeID, p.Message, p.Path, v, p.MinLength)
				}
				if p.Pattern != nil && !p.Pattern.MatchString(v) {
					addErr("[%s] %s: %s (path markov:%s value %q failed pattern %s)",
						shape.Name, nodeID, p.Message, p.Path, v, p.Pattern.String())
				}
				if p.NodeKind == "IRI" && p.Path != "hasSource" {
					targetClass, exists := nodeClass[v]
					if !exists {
						addErr("[%s] %s: unresolved IRI reference %q on path markov:%s",
							shape.Name, nodeID, v, p.Path)
					} else if p.Class != "" && targetClass != p.Class {
						addErr("[%s] %s: path markov:%s references %q of class %s, want %s",
							shape.Name, nodeID, p.Path, v, targetClass, p.Class)
					}
				}
			}
		}
	}

	// 4. Validate SurfaceGraph root node.
	var pkgIDs []string
	for _, p := range nonNilPkgs {
		pkgIDs = append(pkgIDs, p.ID)
	}
	checkShape(g.ID, g.Type, map[string][]string{
		"module":     nonEmpty(g.Module),
		"hasPackage": pkgIDs,
	})

	// 5. Validate each Package, File, Symbol, Source, and Invariant node.
	for _, pkg := range nonNilPkgs {
		var fileIDs []string
		for _, f := range pkg.Files() {
			fileIDs = append(fileIDs, f.ID)
		}
		checkShape(pkg.ID, pkg.Type, map[string][]string{
			"package":      nonEmpty(pkg.Package),
			"importPath":   nonEmpty(pkg.ImportPath),
			"dir":          nonEmpty(pkg.Dir),
			"dependsOn":    pkg.DependsOn,
			"hasFile":      fileIDs,
			"knownUnknown": pkg.KnownUnknowns,
		})

		for _, f := range pkg.Files() {
			var symIDs []string
			for _, s := range f.Symbols {
				symIDs = append(symIDs, s.ID)
			}
			checkShape(f.ID, f.Type, map[string][]string{
				"path":      nonEmpty(f.Path),
				"hasSymbol": symIDs,
			})

			for _, s := range f.Symbols {
				var invIDs []string
				for _, inv := range s.Invariants {
					invIDs = append(invIDs, inv.ID)
				}
				var srcPlaceholders []string
				for i := range s.Src {
					srcPlaceholders = append(srcPlaceholders, fmt.Sprintf("%s#src%d", s.ID, i))
				}

				checkShape(s.ID, s.Type, map[string][]string{
					"name":         nonEmpty(s.Name),
					"receiverType": nonEmpty(s.ReceiverType),
					"signature":    nonEmpty(s.Signature),
					"def":          nonEmpty(s.Def),
					"req":          nonEmpty(s.Req),
					"err":          nonEmpty(s.Err),
					"delegatesTo":  nonEmpty(s.DelegatesTo),
					"references":   s.References,
					"hasSource":    srcPlaceholders,
					"hasInvariant": invIDs,
				})

				// Validate Go signature syntax using go/parser.
				if err := validateGoSignature(s); err != nil {
					addErr("symbol %s has invalid Go signature %q: %v", s.ID, s.Signature, err)
				}

				for idx, src := range s.Src {
					srcID := fmt.Sprintf("%s#src%d", s.ID, idx)
					checkShape(srcID, src.Type, map[string][]string{
						"citation": nonEmpty(src.Citation),
						"url":      nonEmpty(src.URL),
						"title":    nonEmpty(src.Title),
					})
				}

				for _, inv := range s.Invariants {
					checkShape(inv.ID, inv.Type, map[string][]string{
						"name":       nonEmpty(inv.Name),
						"expr":       nonEmpty(inv.Expr),
						"references": inv.References,
					})
				}
			}
		}
	}

	// 6. Verify package dependency graph is a DAG (no cycles).
	if err := checkPackageDAG(nonNilPkgs); err != nil {
		addErr("package dependency cycle: %v", err)
	}

	return errs
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func validateGoSignature(s *SymbolNode) error {
	decl := s.Signature
	if s.Type == "Type" && strings.Contains(decl, "{ ... }") {
		decl = strings.ReplaceAll(decl, "{ ... }", "{}")
	}
	src := "package p\n" + decl + "\n"
	_, err := parser.ParseFile(token.NewFileSet(), "sig.go", src, 0)
	return err
}

func checkPackageDAG(pkgs []*PackageNode) error {
	adj := map[string][]string{}
	for _, p := range pkgs {
		adj[p.ID] = p.DependsOn
	}
	state := map[string]int{} // 0 = unvisited, 1 = visiting, 2 = visited
	var dfs func(u string) error
	dfs = func(u string) error {
		state[u] = 1
		for _, v := range adj[u] {
			if state[v] == 1 {
				return fmt.Errorf("cycle detected involving %s -> %s", u, v)
			}
			if state[v] == 0 {
				if err := dfs(v); err != nil {
					return err
				}
			}
		}
		state[u] = 2
		return nil
	}
	for _, p := range pkgs {
		if state[p.ID] == 0 {
			if err := dfs(p.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ── AST & Policy Verification ─────────────────────────────────────────────────

var (
	reRFC2119  = regexp.MustCompile(`(MUST_NOT|MUST|SHOULD_NOT|SHOULD|SHALL_NOT|SHALL|REQUIRED|RECOMMENDED|OPTIONAL|MAY)([A-Z_]|$)`)
	reLaTeXCmd = regexp.MustCompile(`\\(frac|sum|forall|exists|mathbb|mathcal|mathbf|varepsilon|pi|sigma|gamma|alpha|delta|infty|le|ge|neq|iff|implies|land|lor|subseteq|subset|cup|cap|cdot|times|left|right)\b`)
	reMultiSpc = regexp.MustCompile(`\s+`)
)

// VerifyAST checks implemented Go packages in repoRoot against the surface graph
// and AGENTS.md documentation/testing policies.
// If targetPkgs is empty, all packages whose directory contains doc.go are checked.
func VerifyAST(g *SurfaceGraph, repoRoot string, targetPkgs []string) (int, []error) {
	var errs []error
	addErr := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	allPkgs := []*PackageNode{g.Layer1, g.Reduce, g.Solve, g.Infer, g.Control}
	byName := map[string]*PackageNode{}
	for _, p := range allPkgs {
		if p != nil {
			byName[p.Package] = p
		}
	}

	var toCheck []*PackageNode
	if len(targetPkgs) > 0 {
		for _, raw := range targetPkgs {
			for _, name := range strings.Split(raw, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if name == "layer1" {
					name = "markov"
				}
				p, ok := byName[name]
				if !ok {
					addErr("unknown package %q in --verify-ast", name)
					continue
				}
				toCheck = append(toCheck, p)
			}
		}
	} else {
		for _, p := range allPkgs {
			if p == nil {
				continue
			}
			docPath := filepath.Join(repoRoot, p.Dir, "doc.go")
			if _, err := os.Stat(docPath); err == nil {
				toCheck = append(toCheck, p)
			}
		}
	}

	checkedCount := 0
	for _, pkg := range toCheck {
		checkedCount++
		pkgDir := filepath.Join(repoRoot, pkg.Dir)

		// 1. Check doc.go exists and has package doc comment (AGENTS.md Documentation #1).
		docPath := filepath.Join(pkgDir, "doc.go")
		fset := token.NewFileSet()
		docAST, err := parser.ParseFile(fset, docPath, nil, parser.ParseComments)
		if err != nil {
			addErr("[%s] missing or unparseable doc.go: %v", pkg.Package, err)
		} else {
			if docAST.Name.Name != pkg.Package {
				addErr("[%s] doc.go declares package %q, want %q", pkg.Package, docAST.Name.Name, pkg.Package)
			}
			if docAST.Doc == nil || strings.TrimSpace(docAST.Doc.Text()) == "" {
				addErr("[%s] doc.go MUST have a package doc comment", pkg.Package)
			} else if reLaTeXCmd.MatchString(docAST.Doc.Text()) {
				addErr("[%s] doc.go contains raw LaTeX backslash commands (use Unicode/preformatted blocks per AGENTS.md)", pkg.Package)
			}
		}

		// 2. Check each file and symbol in surface.jsonld.
		for _, fnode := range pkg.Files() {
			if fnode.Path == "doc.go" {
				continue
			}
			fpath := filepath.Join(pkgDir, fnode.Path)
			ffset := token.NewFileSet()
			fileAST, err := parser.ParseFile(ffset, fpath, nil, parser.ParseComments)
			if err != nil {
				addErr("[%s/%s] cannot parse file: %v", pkg.Package, fnode.Path, err)
				continue
			}

			actualSigs := extractExportedDeclarations(ffset, fileAST)

			for _, sym := range fnode.Symbols {
				gotSig, ok := actualSigs[sym.Name]
				if !ok {
					addErr("[%s/%s] missing exported symbol %s (want `%s`)", pkg.Package, fnode.Path, sym.Name, sym.Signature)
					continue
				}
				wantNorm, err := normalizeExpectedSignature(sym)
				if err != nil {
					addErr("[%s/%s] failed to normalize expected signature for %s: %v", pkg.Package, fnode.Path, sym.Name, err)
					continue
				}
				// For struct types whose graph signature ends with "{ ... }" (unexported fields),
				// check prefix "type Name struct".
				if sym.Type == "Type" && strings.Contains(sym.Signature, "{ ... }") {
					if !strings.HasPrefix(gotSig.sig, "type "+sym.Name+" struct") {
						addErr("[%s/%s] symbol %s signature mismatch:\n      got:  %s\n      want: type %s struct { ... }",
							pkg.Package, fnode.Path, sym.Name, gotSig.sig, sym.Name)
					}
				} else if gotSig.sig != wantNorm {
					addErr("[%s/%s] symbol %s signature mismatch:\n      got:  %s\n      want: %s",
						pkg.Package, fnode.Path, sym.Name, gotSig.sig, wantNorm)
				}
				if strings.TrimSpace(gotSig.doc) == "" {
					addErr("[%s/%s] exported symbol %s is missing a Go doc comment (AGENTS.md Documentation #2)", pkg.Package, fnode.Path, sym.Name)
				} else if reLaTeXCmd.MatchString(gotSig.doc) {
					addErr("[%s/%s] exported symbol %s doc comment contains raw LaTeX commands (AGENTS.md Documentation #4)", pkg.Package, fnode.Path, sym.Name)
				}
			}
		}

		// 3. Check test files in pkgDir for RFC 2119 policy (AGENTS.md Tests #5:
		// MUST NOT name test functions using RFC 2119 language) and check Example*
		// functions in example_test.go for doc comments and // Output: blocks.
		entries, err := os.ReadDir(pkgDir)
		if err == nil {
			testFileCount := 0
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
					continue
				}
				testFileCount++
				tfpath := filepath.Join(pkgDir, e.Name())
				tfset := token.NewFileSet()
				tAST, err := parser.ParseFile(tfset, tfpath, nil, parser.ParseComments)
				if err != nil {
					addErr("[%s/%s] cannot parse test file: %v", pkg.Package, e.Name(), err)
					continue
				}
				for _, decl := range tAST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv != nil {
						continue
					}
					if reRFC2119.MatchString(fn.Name.Name) {
						addErr("[%s/%s] test function %s MUST NOT use RFC 2119 keywords in function name (AGENTS.md Tests #5)",
							pkg.Package, e.Name(), fn.Name.Name)
					}
					if strings.HasPrefix(fn.Name.Name, "Example") {
						if fn.Doc == nil || len(strings.TrimSpace(fn.Doc.Text())) < 20 {
							addErr("[%s/%s] %s MUST have a situational-maxim doc comment (AGENTS.md Examples #2)",
								pkg.Package, e.Name(), fn.Name.Name)
						}
						if !hasOutputComment(tfset, tAST, fn) {
							addErr("[%s/%s] %s MUST include a '// Output:' comment block (AGENTS.md Examples #1)",
								pkg.Package, e.Name(), fn.Name.Name)
						}
					}
				}
			}
			if testFileCount == 0 {
				addErr("[%s] package has no *_test.go files", pkg.Package)
			}
		}
	}

	return checkedCount, errs
}

type declInfo struct {
	sig string
	doc string
}

func extractExportedDeclarations(fset *token.FileSet, f *ast.File) map[string]declInfo {
	out := map[string]declInfo{}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			key := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recvType := baseTypeName(d.Recv.List[0].Type)
				if !ast.IsExported(recvType) {
					continue
				}
				key = recvType + "." + d.Name.Name
			}
			cp := *d
			cp.Doc = nil
			cp.Body = nil
			if cp.Recv != nil {
				cp.Recv = stripFieldNames(cp.Recv)
			}
			cp.Type = stripFuncTypeNames(cp.Type)
			out[key] = declInfo{
				sig: formatNode(fset, &cp),
				doc: d.Doc.Text(),
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if !s.Name.IsExported() {
						continue
					}
					doc := d.Doc.Text()
					if s.Doc != nil && s.Doc.Text() != "" {
						doc = s.Doc.Text()
					}
					typeDecl := &ast.GenDecl{
						Tok:   token.TYPE,
						Specs: []ast.Spec{&ast.TypeSpec{Name: s.Name, Type: s.Type}},
					}
					out[s.Name.Name] = declInfo{
						sig: formatNode(fset, typeDecl),
						doc: doc,
					}
				case *ast.ValueSpec:
					if d.Tok != token.VAR {
						continue
					}
					for _, ident := range s.Names {
						if !ident.IsExported() {
							continue
						}
						doc := d.Doc.Text()
						if s.Doc != nil && s.Doc.Text() != "" {
							doc = s.Doc.Text()
						}
						out[ident.Name] = declInfo{
							sig: "var " + ident.Name + " error",
							doc: doc,
						}
					}
				}
			}
		}
	}
	return out
}

func normalizeExpectedSignature(s *SymbolNode) (string, error) {
	if s.Type == "Error" {
		return s.Signature, nil
	}
	decl := s.Signature
	if s.Type == "Type" && strings.Contains(decl, "{ ... }") {
		decl = strings.ReplaceAll(decl, "{ ... }", "{}")
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sig.go", "package p\n"+decl+"\n", 0)
	if err != nil {
		return "", err
	}
	decls := extractExportedDeclarations(fset, f)
	info, ok := decls[s.Name]
	if !ok {
		return "", fmt.Errorf("symbol %s not found in parsed signature %q", s.Name, s.Signature)
	}
	return info.sig, nil
}

func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.Ident:
		return t.Name
	default:
		return ""
	}
}

func stripFieldNames(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	var list []*ast.Field
	for _, field := range fl.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			list = append(list, &ast.Field{Type: field.Type})
		}
	}
	return &ast.FieldList{List: list}
}

func stripFuncTypeNames(ft *ast.FuncType) *ast.FuncType {
	cp := *ft
	cp.Params = stripFieldNames(ft.Params)
	cp.Results = stripFieldNames(ft.Results)
	return &cp
}

func formatNode(fset *token.FileSet, node any) string {
	var buf bytes.Buffer
	_ = format.Node(&buf, fset, node)
	s := strings.ReplaceAll(buf.String(), "\n", "; ")
	s = reMultiSpc.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "{; ", "{ ")
	s = strings.ReplaceAll(s, "; }", " }")
	return strings.TrimSpace(s)
}

func hasOutputComment(fset *token.FileSet, f *ast.File, fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	start := fn.Body.Pos()
	end := fn.Body.End()
	for _, cg := range f.Comments {
		if cg.Pos() >= start && cg.End() <= end {
			for _, c := range cg.List {
				txt := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
				if strings.HasPrefix(txt, "Output:") || strings.HasPrefix(txt, "Unordered output:") {
					return true
				}
			}
		}
	}
	return false
}

// ── Persona Document Validation (docs/personas/persona.shacl.ttl) ─────────────

var requiredPersonaSections = []string{
	"Skill Stage",
	"Prior Knowledge",
	"Mental Model & Vocabulary Bridge",
	"Situational Maxims",
	"Tone, Do's & Don'ts",
	"Examples Roadmap",
	"Sources",
}

var requiredRunbookSections = []string{
	"When to Use This Runbook",
	"Preconditions",
	"Step-by-Step Procedure",
	"Verification",
	"Troubleshooting & Rollback",
	"Runbook Template",
}

// ValidatePersonas validates all Markdown documents in personasDir against
// personaSHACLPath (checking OKF type: persona frontmatter and required sections).
func ValidatePersonas(personasDir, personaSHACLPath string) (int, []error) {
	return validateTypedMarkdownDir(personasDir, personaSHACLPath, "Persona", "urn:markov:persona:", requiredPersonaSections)
}

// ValidateRunbooks validates all Markdown documents in runbooksDir against
// runbookSHACLPath (checking OKF type: runbook frontmatter and required sections).
func ValidateRunbooks(runbooksDir, runbookSHACLPath string) (int, []error) {
	return validateTypedMarkdownDir(runbooksDir, runbookSHACLPath, "Runbook", "urn:markov:runbook:", requiredRunbookSections)
}

func validateTypedMarkdownDir(dirPath, shaclPath, targetClass, iriPrefix string, requiredSections []string) (int, []error) {
	var errs []error
	addErr := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return 0, nil
	}
	ttlBytes, err := os.ReadFile(shaclPath)
	if err != nil {
		addErr("read %s: %v", shaclPath, err)
		return 0, errs
	}
	shapes, err := ParseSHACLShapes(string(ttlBytes))
	if err != nil {
		addErr("parse %s: %v", shaclPath, err)
		return 0, errs
	}
	shape, ok := shapes[targetClass]
	if !ok {
		addErr("%s: missing NodeShape for targetClass markov:%s", shaclPath, targetClass)
		return 0, errs
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		addErr("read dir %s: %v", dirPath, err)
		return 0, errs
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		count++
		slug := strings.TrimSuffix(e.Name(), ".md")
		nodeID := iriPrefix + slug
		rawBytes, err := os.ReadFile(filepath.Join(dirPath, e.Name()))
		if err != nil {
			addErr("read %s: %v", e.Name(), err)
			continue
		}
		fm, headings, err := parseMarkdownFrontmatterAndHeadings(string(rawBytes))
		if err != nil {
			addErr("[%s] %v", e.Name(), err)
			continue
		}

		var matchedSections []string
		for _, reqSec := range requiredSections {
			for _, h := range headings {
				if strings.Contains(h, reqSec) {
					matchedSections = append(matchedSections, reqSec)
					break
				}
			}
		}

		props := map[string][]string{
			"type":        nonEmpty(fm["type"]),
			"title":       nonEmpty(fm["title"]),
			"description": nonEmpty(fm["description"]),
			"stage":       nonEmpty(fm["stage"]),
			"timestamp":   nonEmpty(fm["timestamp"]),
			"hasSection":  matchedSections,
		}

		if shape.Pattern != nil && !shape.Pattern.MatchString(nodeID) {
			addErr("[%s] IRI %q does not match %s", shape.Name, nodeID, shape.Pattern.String())
		}
		for _, p := range shape.Properties {
			vals := props[p.Path]
			if p.MinCount > 0 && len(vals) < p.MinCount {
				addErr("[%s] %s: %s (path markov:%s got %d, minCount %d)",
					shape.Name, e.Name(), p.Message, p.Path, len(vals), p.MinCount)
				continue
			}
			if p.MaxCount >= 0 && len(vals) > p.MaxCount {
				addErr("[%s] %s: %s (path markov:%s got %d, maxCount %d)",
					shape.Name, e.Name(), p.Message, p.Path, len(vals), p.MaxCount)
				continue
			}
			for _, v := range vals {
				if p.MinLength > 0 && len(v) < p.MinLength {
					addErr("[%s] %s: %s (path markov:%s value %q < minLength %d)",
						shape.Name, e.Name(), p.Message, p.Path, v, p.MinLength)
				}
				if p.Pattern != nil && !p.Pattern.MatchString(v) {
					addErr("[%s] %s: %s (path markov:%s value %q failed pattern %s)",
						shape.Name, e.Name(), p.Message, p.Path, v, p.Pattern.String())
				}
			}
		}
	}

	if count == 0 {
		addErr("no .md files found in %s", dirPath)
	}
	return count, errs
}

// ── Documentation Renderer & Symbol Inspector (from surface.jsonld) ───────────

type pkgDisplayMeta struct {
	Heading string
	Summary string
	Node    *PackageNode
}

func packageDisplayList(g *SurfaceGraph) []pkgDisplayMeta {
	return []pkgDisplayMeta{
		{
			Heading: "Layer 1: Core Primitives (`package markov`)",
			Summary: "Row-stochastic transition matrices (`Kernel`), probability simplex vectors (`Distribution`), and state-space equivalence classes (`Partition`).",
			Node:    g.Layer1,
		},
		{
			Heading: "Layer 2: Model Order Reduction (`package reduce`)",
			Summary: "Temporal censoring via Schur complement (`Trace`, `TraceReward`), spatial quotienting via Kemeny–Snell projection (`Lump`, `LumpabilityResidual`), and Paige–Tarjan bisimulation (`CoarsestPartition`, `AreBisimilar`).",
			Node:    g.Reduce,
		},
		{
			Heading: "Layer 2: Exact Dynamics & Structural Solvers (`package solve`)",
			Summary: "Tarjan SCC & periodicity classification (`ClassifyStates`), subtraction-free Grassmann–Taksar–Heyman steady-state solver (`Stationary`), fundamental matrices (`FundamentalMatrix`, `AbsorbingFundamental`), mean first-passage times (`MFPT`, `MFPTToTarget`, `MFPTDelta`), and Shannon/KL entropy rates (`EntropyRate`, `KLDivergenceRate`).",
			Node:    g.Solve,
		},
		{
			Heading: "Layer 2: Statistical Inference (`package infer`)",
			Summary: "Dirichlet-multinomial online transition count estimation with exponential moving average decay (`Estimator`, `FromTransitions`, `FromSequence`) and Anderson–Goodman / Billingsley log-likelihood ratio order testing (`MarkovOrderTest`).",
			Node:    g.Infer,
		},
		{
			Heading: "Layer 2: Optimal Control & MDPs (`package control`)",
			Summary: "Discounted Markov Decision Processes (`MDP`, `InducedKernel`), Bellman optimality via `ValueIteration` and Howard's `PolicyIteration`, and Givan–Dean–Greig reward-aware `MDPBisimulation`.",
			Node:    g.Control,
		},
	}
}

// RenderArchitectureMarkdown renders docs/architecture.md deterministically from
// docs/surface/surface.jsonld, including a Mermaid package/operator graph,
// GitHub-flavored LaTeX reference tables per package, and the machine-parseable
// ASCII symbol tree.
func RenderArchitectureMarkdown(g *SurfaceGraph) string {
	var b strings.Builder

	b.WriteString("# `markov` Architecture & Mathematical Knowledge Graph\n\n")
	b.WriteString("> **Generated from [`docs/surface/surface.jsonld`](./surface/surface.jsonld)** — Do not edit by hand; run `make docs-gen` (`go run ./tools/surface render`) to regenerate.\n\n")
	b.WriteString("This document defines the complete public API surface, mathematical definitions, domain preconditions, boundary conditions, primary literature sources, and algebraic invariants across `" + g.Module + "`.\n\n")

	b.WriteString("## Graph Predicate Schema\n\n")
	b.WriteString("Each exported symbol node in [`docs/surface/surface.jsonld`](./surface/surface.jsonld) uses a fixed set of typed predicates validated by [`docs/surface/surface.shacl.ttl`](./surface/surface.shacl.ttl) and checked against the Go AST via `make surface-ast`:\n\n")
	b.WriteString("* **`def:`** — Formal mathematical definition of the operator or object.\n")
	b.WriteString("* **`req:`** — Domain preconditions required for the operator to be well-defined.\n")
	b.WriteString("* **`err:`** — Explicit error or sentinel behavior when preconditions are violated.\n")
	b.WriteString("* **`src:`** — Verifiable, open-access primary literature or reference implementation.\n")
	b.WriteString("* **`⊢`** — Algebraic or probabilistic invariants that **MUST** hold within tolerance $\\varepsilon = \\text{tol}$.\n\n")

	b.WriteString("## Package & Operator Architecture Graph\n\n")
	b.WriteString("```mermaid\n")
	b.WriteString("graph TD\n")
	b.WriteString("    subgraph layer1[\"Layer 1: package markov\"]\n")
	b.WriteString("        Kernel[\"Kernel (P ∈ ℝⁿˣⁿ)\"]\n")
	b.WriteString("        Distribution[\"Distribution (π ∈ Δⁿ⁻¹)\"]\n")
	b.WriteString("        Partition[\"Partition (ℬ = {B₀..Bₘ₋₁})\"]\n")
	b.WriteString("    end\n")
	b.WriteString("    subgraph reduce[\"Layer 2: package reduce\"]\n")
	b.WriteString("        Trace[\"Trace: P_A = a + b(I - c)⁻¹d\"]\n")
	b.WriteString("        Lump[\"Lump: P̂ = U P V\"]\n")
	b.WriteString("        Bisim[\"CoarsestPartition / AreBisimilar\"]\n")
	b.WriteString("    end\n")
	b.WriteString("    subgraph solve[\"Layer 2: package solve\"]\n")
	b.WriteString("        Classify[\"ClassifyStates (Tarjan SCCs & Periods)\"]\n")
	b.WriteString("        Stationary[\"Stationary (GTH Solver: πP = π)\"]\n")
	b.WriteString("        Fund[\"FundamentalMatrix Z & AbsorbingFundamental N\"]\n")
	b.WriteString("        MFPT[\"MFPT / MFPTToTarget / MFPTDelta\"]\n")
	b.WriteString("        Entropy[\"EntropyRate H(P) & KLDivergenceRate D_KL(P‖Q)\"]\n")
	b.WriteString("    end\n")
	b.WriteString("    subgraph infer[\"Layer 2: package infer\"]\n")
	b.WriteString("        Estimator[\"Estimator / FromSequence (Dirichlet α + EMA γ)\"]\n")
	b.WriteString("        OrderTest[\"MarkovOrderTest (G² ~ χ²)\"]\n")
	b.WriteString("    end\n")
	b.WriteString("    subgraph control[\"Layer 2: package control\"]\n")
	b.WriteString("        MDP[\"MDP (S, A, {Pᵃ}, R, γ)\"]\n")
	b.WriteString("        DP[\"ValueIteration / PolicyIteration\"]\n")
	b.WriteString("        MDPBisim[\"MDPBisimulation (Givan-Dean-Greig)\"]\n")
	b.WriteString("    end\n")
	b.WriteString("    Kernel --> Trace & Lump & Bisim\n")
	b.WriteString("    Partition --> Lump & Bisim\n")
	b.WriteString("    Distribution --> Lump\n")
	b.WriteString("    Kernel --> Classify --> Stationary --> Fund --> MFPT\n")
	b.WriteString("    Stationary --> Entropy\n")
	b.WriteString("    Estimator --> Kernel\n")
	b.WriteString("    Kernel --> MDP --> DP & MDPBisim\n")
	b.WriteString("    Bisim --> MDPBisim\n")
	b.WriteString("```\n\n")

	for _, pm := range packageDisplayList(g) {
		if pm.Node == nil {
			continue
		}
		b.WriteString("---\n\n")
		b.WriteString("## " + pm.Heading + "\n\n")
		b.WriteString(pm.Summary + "\n\n")

		// Render Mathematical Reference Table for Functions & Methods.
		b.WriteString("### Mathematical Operator & Invariant Table\n\n")
		b.WriteString("| Symbol & File | Definition (`def`) | Preconditions (`req`) & Errors (`err`) | Algebraic Invariants (`⊢`) & Sources (`src`) |\n")
		b.WriteString("| :--- | :--- | :--- | :--- |\n")
		for _, f := range pm.Node.Files() {
			for _, s := range f.Symbols {
				if s.Type != "Function" && s.Type != "Method" {
					continue
				}
				symCol := fmt.Sprintf("**`%s`**<br>`%s`<br><sub>`%s`</sub>", s.Name, escapeTableCell(s.Signature), f.Path)
				defCol := escapeTableCell(s.Def)
				var reqParts []string
				if s.Req != "" {
					reqParts = append(reqParts, "**req:** "+escapeTableCell(s.Req))
				}
				if s.Err != "" {
					reqParts = append(reqParts, "**err:** "+escapeTableCell(s.Err))
				}
				reqCol := "—"
				if len(reqParts) > 0 {
					reqCol = strings.Join(reqParts, "<br>")
				}
				var invParts []string
				for _, inv := range s.Invariants {
					invParts = append(invParts, fmt.Sprintf("⊢ **%s:** %s", escapeTableCell(inv.Name), escapeTableCell(inv.Expr)))
				}
				for _, src := range s.Src {
					invParts = append(invParts, fmt.Sprintf("📄 [%s](%s)", escapeTableCell(src.Citation), src.URL))
				}
				invCol := "—"
				if len(invParts) > 0 {
					invCol = strings.Join(invParts, "<br>")
				}
				b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", symCol, defCol, reqCol, invCol))
			}
		}
		b.WriteString("\n")

		// Render canonical ASCII tree block (round-trip compatible with ParseArchitectureMarkdown).
		b.WriteString("<details>\n<summary><strong>Surface Knowledge Graph Tree (<code>" + pm.Node.Package + "/</code>)</strong></summary>\n\n")
		b.WriteString("```text\n")
		renderPackageTree(&b, pm.Node)
		b.WriteString("```\n\n</details>\n\n")
	}

	renderRelationalIndices(&b, g)

	return b.String()
}

func renderRelationalIndices(b *strings.Builder, g *SurfaceGraph) {
	pkgs := []*PackageNode{g.Layer1, g.Reduce, g.Solve, g.Infer, g.Control}

	// 1. Sentinel Error Boundary Matrix (reverse index from Error IRI -> all symbols referencing it).
	b.WriteString("---\n\n")
	b.WriteString("## Cross-Package Relational Indices\n\n")
	b.WriteString("### 1. Sentinel Error Boundary Matrix\n\n")
	b.WriteString("Reverse graph traversal (`?symbol markov:references ?err`) showing every domain boundary condition across the library that returns each exported sentinel error:\n\n")
	b.WriteString("| Sentinel Error | Returning Symbol | Boundary Condition (`err` / `req`) |\n")
	b.WriteString("| :--- | :--- | :--- |\n")
	if g.Layer1 != nil && g.Layer1.Errors != nil {
		for _, errSym := range g.Layer1.Errors.Symbols {
			matchedAny := false
			for _, pkg := range pkgs {
				if pkg == nil {
					continue
				}
				for _, f := range pkg.Files() {
					for _, s := range f.Symbols {
						for _, ref := range s.References {
							if ref == errSym.ID {
								matchedAny = true
								b.WriteString(fmt.Sprintf("| `%s` | `%s.%s` | %s |\n",
									errSym.Name, pkg.Package, s.Name, escapeTableCell(s.Err)))
							}
						}
					}
				}
			}
			if !matchedAny {
				b.WriteString(fmt.Sprintf("| `%s` | — | Reserved sentinel error |\n", errSym.Name))
			}
		}
	}
	b.WriteString("\n")

	// 2. Cross-Symbol Algebraic Laws (invariants whose `references` edge links to external symbols).
	b.WriteString("### 2. Compositional & Cross-Symbol Algebraic Laws\n\n")
	b.WriteString("Graph traversal (`?inv markov:references ?targetSymbol`) surfacing algebraic laws that couple multiple operators across files or packages:\n\n")
	b.WriteString("| Owner Symbol | Invariant (`⊢`) | Formula (`expr`) | Coupled Symbols (`references`) |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files() {
			for _, s := range f.Symbols {
				for _, inv := range s.Invariants {
					var extRefs []string
					for _, r := range inv.References {
						if r != s.ID {
							extRefs = append(extRefs, "`"+strings.TrimPrefix(r, "urn:markov:sym:")+"`")
						}
					}
					if len(extRefs) > 0 {
						b.WriteString(fmt.Sprintf("| `%s.%s` | **%s** | %s | %s |\n",
							pkg.Package, s.Name, escapeTableCell(inv.Name), escapeTableCell(inv.Expr), strings.Join(extRefs, ", ")))
					}
				}
			}
		}
	}
	b.WriteString("\n")

	// 3. Literature-to-Operator Citation Index.
	b.WriteString("### 3. Primary Literature-to-Operator Index\n\n")
	b.WriteString("Reverse citation index (`?symbol markov:hasSource ?src`) mapping foundational papers and textbooks to the Go symbols that implement them:\n\n")
	b.WriteString("| Primary Source | URL | Implemented Symbols |\n")
	b.WriteString("| :--- | :--- | :--- |\n")
	type srcGroup struct {
		citation string
		url      string
		symbols  []string
	}
	var srcOrder []*srcGroup
	byCitation := map[string]*srcGroup{}
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files() {
			for _, s := range f.Symbols {
				for _, src := range s.Src {
					key := src.Citation + "|" + src.URL
					grp, exists := byCitation[key]
					if !exists {
						grp = &srcGroup{citation: src.Citation, url: src.URL}
						byCitation[key] = grp
						srcOrder = append(srcOrder, grp)
					}
					grp.symbols = append(grp.symbols, fmt.Sprintf("`%s.%s`", pkg.Package, s.Name))
				}
			}
		}
	}
	for _, grp := range srcOrder {
		b.WriteString(fmt.Sprintf("| %s | [link](%s) | %s |\n",
			escapeTableCell(grp.citation), grp.url, strings.Join(grp.symbols, ", ")))
	}
	b.WriteString("\n")
}

func escapeTableCell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

func renderPackageTree(b *strings.Builder, pkg *PackageNode) {
	b.WriteString(pkg.Package + "/\n")
	if pkg.Package == "markov" {
		b.WriteString("├── go.mod / go.sum\n")
		b.WriteString("├── Makefile\n")
		b.WriteString("├── LICENSE\n")
		b.WriteString("├── README.md\n")
		b.WriteString("├── AGENTS.md\n")
	}
	files := pkg.Files()
	for _, f := range files {
		b.WriteString("├── " + f.Path + "\n")
		for _, s := range f.Symbols {
			b.WriteString("│   ├── " + s.Signature + "\n")

			var preds []string
			if s.Def != "" {
				preds = append(preds, "def: "+s.Def)
			}
			if s.Req != "" {
				preds = append(preds, "req: "+s.Req)
			}
			if s.Err != "" {
				preds = append(preds, "err: "+s.Err)
			}
			if len(s.Src) > 0 {
				var links []string
				for _, src := range s.Src {
					if src.Title != "" {
						links = append(links, fmt.Sprintf("[%s](%s %q)", src.Citation, src.URL, src.Title))
					} else {
						links = append(links, fmt.Sprintf("[%s](%s)", src.Citation, src.URL))
					}
				}
				preds = append(preds, "src: "+strings.Join(links, " & "))
			}
			for _, inv := range s.Invariants {
				preds = append(preds, fmt.Sprintf("⊢ %s: %s", inv.Name, inv.Expr))
			}

			for pIdx, pred := range preds {
				pBranch := "├── "
				if pIdx == len(preds)-1 {
					pBranch = "└── "
				}
				b.WriteString("│   │   " + pBranch + pred + "\n")
			}
		}
	}
	b.WriteString("└── known_unknowns.list\n")
	for uIdx, u := range pkg.KnownUnknowns {
		uBranch := "├── "
		if uIdx == len(pkg.KnownUnknowns)-1 {
			uBranch = "└── "
		}
		b.WriteString("    " + uBranch + u + "\n")
	}
}

// InspectSymbols prints a human-readable mathematical specification card for all
// symbols in g whose ID, name, signature, or file path matches pattern (case-insensitive),
// including reverse graph traversals (inbound references, delegations, and cross-package
// invariants that couple with the symbol) which `go doc` cannot compute.
func InspectSymbols(g *SurfaceGraph, pattern string, w io.Writer) int {
	q := strings.ToLower(strings.TrimSpace(pattern))
	pkgs := []*PackageNode{g.Layer1, g.Reduce, g.Solve, g.Infer, g.Control}

	// Build reverse indices across the entire graph:
	// - inboundRefs[targetID] -> list of symbols referencing targetID
	// - inboundDelegations[targetID] -> list of symbols delegating to targetID
	// - coupledInvariants[targetID] -> list of (ownerSymbolID, InvariantNode) referencing targetID
	inboundRefs := map[string][]string{}
	inboundDelegations := map[string][]string{}
	type coupledInv struct {
		ownerID string
		inv     *InvariantNode
	}
	coupledInvariants := map[string][]coupledInv{}

	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files() {
			for _, s := range f.Symbols {
				if s.DelegatesTo != "" {
					inboundDelegations[s.DelegatesTo] = append(inboundDelegations[s.DelegatesTo], s.ID)
				}
				for _, r := range s.References {
					inboundRefs[r] = append(inboundRefs[r], s.ID)
				}
				for _, inv := range s.Invariants {
					for _, r := range inv.References {
						if r != s.ID {
							coupledInvariants[r] = append(coupledInvariants[r], coupledInv{ownerID: s.ID, inv: inv})
						}
					}
				}
			}
		}
	}

	matched := 0
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.Files() {
			fileRel := pkg.Package + "/" + f.Path
			if pkg.Package == "markov" {
				fileRel = f.Path
			}
			for _, s := range f.Symbols {
				haystack := strings.ToLower(s.ID + " " + s.Name + " " + s.Signature + " " + fileRel)
				if !strings.Contains(haystack, q) {
					continue
				}
				matched++
				fmt.Fprintf(w, "─── %s (%s) ───\n", s.ID, s.Type)
				fmt.Fprintf(w, "  File:        %s\n", fileRel)
				fmt.Fprintf(w, "  Signature:   %s\n", s.Signature)
				if s.Def != "" {
					fmt.Fprintf(w, "  Definition:  %s\n", s.Def)
				}
				if s.Req != "" {
					fmt.Fprintf(w, "  Requires:    %s\n", s.Req)
				}
				if s.Err != "" {
					fmt.Fprintf(w, "  Errors:      %s\n", s.Err)
				}
				if s.DelegatesTo != "" {
					fmt.Fprintf(w, "  DelegatesTo: %s\n", s.DelegatesTo)
				}
				if len(inboundDelegations[s.ID]) > 0 {
					fmt.Fprintf(w, "  DelegatedBy: %s\n", strings.Join(inboundDelegations[s.ID], ", "))
				}
				if len(s.References) > 0 {
					fmt.Fprintf(w, "  OutboundRefs:%s\n", " "+strings.Join(s.References, ", "))
				}
				if len(inboundRefs[s.ID]) > 0 {
					fmt.Fprintf(w, "  InboundRefs: %s\n", strings.Join(inboundRefs[s.ID], ", "))
				}
				if len(s.Src) > 0 {
					fmt.Fprintf(w, "  Sources:\n")
					for _, src := range s.Src {
						fmt.Fprintf(w, "    • %s (%s)\n", src.Citation, src.URL)
					}
				}
				if len(s.Invariants) > 0 {
					fmt.Fprintf(w, "  Own Invariants:\n")
					for _, inv := range s.Invariants {
						fmt.Fprintf(w, "    ⊢ %s: %s\n", inv.Name, inv.Expr)
					}
				}
				if len(coupledInvariants[s.ID]) > 0 {
					fmt.Fprintf(w, "  Cross-Symbol Laws Referencing %s:\n", s.Name)
					for _, ci := range coupledInvariants[s.ID] {
						fmt.Fprintf(w, "    ⇄ [%s] ⊢ %s: %s\n", strings.TrimPrefix(ci.ownerID, "urn:markov:sym:"), ci.inv.Name, ci.inv.Expr)
					}
				}
				fmt.Fprintln(w)
			}
		}
	}
	if matched == 0 {
		fmt.Fprintf(w, "FAIL  no symbols matching %q in surface graph\n", pattern)
		return 1
	}
	fmt.Fprintf(w, "PASS  %d symbol(s) matched %q\n", matched, pattern)
	return 0
}

func parseMarkdownFrontmatterAndHeadings(md string) (map[string]string, []string, error) {
	lines := strings.Split(md, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, nil, fmt.Errorf("missing YAML frontmatter block ('---')")
	}
	fm := map[string]string{}
	endFM := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endFM = i
			break
		}
		if colon := strings.Index(lines[i], ":"); colon > 0 {
			k := strings.TrimSpace(lines[i][:colon])
			v := strings.TrimSpace(lines[i][colon+1:])
			fm[k] = v
		}
	}
	if endFM < 0 {
		return nil, nil, fmt.Errorf("unterminated YAML frontmatter block")
	}
	var headings []string
	inCode := false
	for _, line := range lines[endFM+1:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if !inCode && strings.HasPrefix(trimmed, "## ") {
			headings = append(headings, strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
		}
	}
	return fm, headings, nil
}

// ── CLI Runner ────────────────────────────────────────────────────────────────

type options struct {
	archPath    string
	surfacePath string
	contextPath string
	shaclPath   string
	repoRoot    string
	emit        bool
	renderDocs  bool
	inspect     string
	checkSync   bool
	verifyAST   bool
	astPkgs     string
}

func run(opts options, out io.Writer) int {
	// If --inspect is passed, load surface.jsonld and print matching symbol cards.
	if opts.inspect != "" {
		surfaceBytes, err := os.ReadFile(opts.surfacePath)
		if err != nil {
			fmt.Fprintf(out, "FAIL  read %s: %v\n", opts.surfacePath, err)
			return 1
		}
		var diskGraph SurfaceGraph
		if err := json.Unmarshal(surfaceBytes, &diskGraph); err != nil {
			fmt.Fprintf(out, "FAIL  unmarshal %s: %v\n", opts.surfacePath, err)
			return 1
		}
		return InspectSymbols(&diskGraph, opts.inspect, out)
	}

	// If --render-docs is passed, render docs/architecture.md from docs/surface/surface.jsonld.
	if opts.renderDocs {
		surfaceBytes, err := os.ReadFile(opts.surfacePath)
		if err != nil {
			fmt.Fprintf(out, "FAIL  read %s: %v\n", opts.surfacePath, err)
			return 1
		}
		var diskGraph SurfaceGraph
		if err := json.Unmarshal(surfaceBytes, &diskGraph); err != nil {
			fmt.Fprintf(out, "FAIL  unmarshal %s: %v\n", opts.surfacePath, err)
			return 1
		}
		rendered := RenderArchitectureMarkdown(&diskGraph)
		if err := os.WriteFile(opts.archPath, []byte(rendered), 0o644); err != nil {
			fmt.Fprintf(out, "FAIL  write %s: %v\n", opts.archPath, err)
			return 1
		}
		fmt.Fprintf(out, "PASS  rendered %s from %s\n", opts.archPath, opts.surfacePath)
	}

	archBytes, err := os.ReadFile(opts.archPath)
	if err != nil {
		fmt.Fprintf(out, "FAIL  read %s: %v\n", opts.archPath, err)
		return 1
	}

	parsedGraph, err := ParseArchitectureMarkdown(string(archBytes))
	if err != nil {
		fmt.Fprintf(out, "FAIL  parse %s: %v\n", opts.archPath, err)
		return 1
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(parsedGraph); err != nil {
		fmt.Fprintf(out, "FAIL  marshal surface graph: %v\n", err)
		return 1
	}
	generatedJSON := buf.Bytes()

	if opts.emit {
		if err := os.WriteFile(opts.surfacePath, generatedJSON, 0o644); err != nil {
			fmt.Fprintf(out, "FAIL  write %s: %v\n", opts.surfacePath, err)
			return 1
		}
	}

	surfaceBytes, err := os.ReadFile(opts.surfacePath)
	if err != nil {
		fmt.Fprintf(out, "FAIL  read %s: %v\n", opts.surfacePath, err)
		return 1
	}

	if opts.checkSync && !bytes.Equal(bytes.TrimSpace(surfaceBytes), bytes.TrimSpace(generatedJSON)) {
		fmt.Fprintf(out, "FAIL  %s is out of sync with %s (run `make docs-gen` or `go run ./tools/surface emit`)\n", opts.archPath, opts.surfacePath)
		return 1
	}

	var diskGraph SurfaceGraph
	if err := json.Unmarshal(surfaceBytes, &diskGraph); err != nil {
		fmt.Fprintf(out, "FAIL  unmarshal %s: %v\n", opts.surfacePath, err)
		return 1
	}

	ctxBytes, err := os.ReadFile(opts.contextPath)
	if err != nil {
		fmt.Fprintf(out, "FAIL  read %s: %v\n", opts.contextPath, err)
		return 1
	}

	shaclBytes, err := os.ReadFile(opts.shaclPath)
	if err != nil {
		fmt.Fprintf(out, "FAIL  read %s: %v\n", opts.shaclPath, err)
		return 1
	}

	errs := Validate(&diskGraph, string(ctxBytes), string(shaclBytes))
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(out, "  ✗  %v\n", e)
		}
		fmt.Fprintf(out, "\nFAIL  %d SHACL/surface validation error(s)\n", len(errs))
		return 1
	}

	pkgs, files, syms, invs, srcs := countStats(&diskGraph)
	fmt.Fprintf(out, "PASS  surface graph valid (%d packages, %d files, %d symbols, %d invariants, %d sources)\n",
		pkgs, files, syms, invs, srcs)

	root := opts.repoRoot
	if root == "" {
		root = "."
	}
	personasDir := filepath.Join(root, "docs", "personas")
	personaSHACL := filepath.Join(personasDir, "persona.shacl.ttl")
	if pCount, pErrs := ValidatePersonas(personasDir, personaSHACL); len(pErrs) > 0 {
		for _, e := range pErrs {
			fmt.Fprintf(out, "  ✗  %v\n", e)
		}
		fmt.Fprintf(out, "\nFAIL  %d persona SHACL validation error(s)\n", len(pErrs))
		return 1
	} else if pCount > 0 {
		fmt.Fprintf(out, "PASS  persona docs valid (%d persona(s) verified against persona.shacl.ttl)\n", pCount)
	}

	runbooksDir := filepath.Join(root, "docs", "runbooks")
	runbookSHACL := filepath.Join(runbooksDir, "runbook.shacl.ttl")
	if rCount, rErrs := ValidateRunbooks(runbooksDir, runbookSHACL); len(rErrs) > 0 {
		for _, e := range rErrs {
			fmt.Fprintf(out, "  ✗  %v\n", e)
		}
		fmt.Fprintf(out, "\nFAIL  %d runbook SHACL validation error(s)\n", len(rErrs))
		return 1
	} else if rCount > 0 {
		fmt.Fprintf(out, "PASS  runbook docs valid (%d runbook(s) verified against runbook.shacl.ttl)\n", rCount)
	}

	if opts.verifyAST {
		var targetPkgs []string
		if opts.astPkgs != "" {
			targetPkgs = strings.Split(opts.astPkgs, ",")
		}
		checked, astErrs := VerifyAST(&diskGraph, root, targetPkgs)
		if len(astErrs) > 0 {
			for _, e := range astErrs {
				fmt.Fprintf(out, "  ✗  %v\n", e)
			}
			fmt.Fprintf(out, "\nFAIL  %d AST/policy verification error(s) across %d package(s)\n", len(astErrs), checked)
			return 1
		}
		fmt.Fprintf(out, "PASS  AST & policy verified across %d implemented package(s)\n", checked)
	}
	return 0
}

func countStats(g *SurfaceGraph) (pkgs, files, syms, invs, srcs int) {
	for _, p := range []*PackageNode{g.Layer1, g.Reduce, g.Solve, g.Infer, g.Control} {
		if p == nil {
			continue
		}
		pkgs++
		for _, f := range p.Files() {
			files++
			for _, s := range f.Symbols {
				syms++
				invs += len(s.Invariants)
				srcs += len(s.Src)
			}
		}
	}
	return
}

// NewRootCommand builds the Cobra CLI for surface validation, inspection, AST
// verification, and documentation generation.
//
// Subcommands:
//
//	surface validate           # Validate surface.jsonld, SHACL shapes, personas, and runbooks (default)
//	surface ast [pkg...]       # Verify implemented Go packages via go/ast against surface.jsonld & AGENTS.md
//	surface inspect <pattern>  # Print mathematical specification & reverse-graph card for matching symbols
//	surface render             # Render docs/architecture.md from docs/surface/surface.jsonld and validate
//	surface emit               # Regenerate docs/surface/surface.jsonld from docs/architecture.md and validate
func NewRootCommand(out io.Writer) *cobra.Command {
	var opts options

	root := &cobra.Command{
		Use:           "surface",
		Short:         "Validate, inspect, and render the markov mathematical surface graph",
		Long:          "Validates docs/surface/surface.jsonld against W3C SHACL shapes and Go AST declarations, queries relational symbol cards, and renders docs/architecture.md.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if code := run(opts, out); code != 0 {
				return fmt.Errorf("surface check failed")
			}
			return nil
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&opts.archPath, "arch", "docs/architecture.md", "path to architecture markdown")
	pf.StringVar(&opts.surfacePath, "surface", "docs/surface/surface.jsonld", "path to JSON-LD surface graph")
	pf.StringVar(&opts.contextPath, "context", "docs/surface/context.jsonld", "path to JSON-LD context")
	pf.StringVar(&opts.shaclPath, "shacl", "docs/surface/surface.shacl.ttl", "path to SHACL shapes TTL")
	pf.StringVar(&opts.repoRoot, "root", ".", "repository root directory")
	pf.BoolVar(&opts.checkSync, "check-sync", true, "verify surface.jsonld matches docs/architecture.md")

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate surface.jsonld, SHACL shapes, personas, and runbooks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if code := run(opts, out); code != 0 {
				return fmt.Errorf("surface validate failed")
			}
			return nil
		},
	}

	astCmd := &cobra.Command{
		Use:   "ast [pkg...]",
		Short: "Verify implemented Go packages against surface.jsonld and AGENTS.md policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := opts
			o.verifyAST = true
			if len(args) > 0 {
				o.astPkgs = strings.Join(args, ",")
			}
			if code := run(o, out); code != 0 {
				return fmt.Errorf("surface ast failed")
			}
			return nil
		},
	}

	inspectCmd := &cobra.Command{
		Use:   "inspect <pattern>",
		Short: "Inspect mathematical specification and reverse-graph relations for symbols matching <pattern>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o := opts
			o.inspect = args[0]
			if code := run(o, out); code != 0 {
				return fmt.Errorf("surface inspect failed")
			}
			return nil
		},
	}

	renderCmd := &cobra.Command{
		Use:   "render",
		Short: "Render docs/architecture.md from docs/surface/surface.jsonld and validate",
		RunE: func(cmd *cobra.Command, _ []string) error {
			o := opts
			o.renderDocs = true
			if code := run(o, out); code != 0 {
				return fmt.Errorf("surface render failed")
			}
			return nil
		},
	}

	emitCmd := &cobra.Command{
		Use:   "emit",
		Short: "Regenerate docs/surface/surface.jsonld from docs/architecture.md and validate",
		RunE: func(cmd *cobra.Command, _ []string) error {
			o := opts
			o.emit = true
			if code := run(o, out); code != 0 {
				return fmt.Errorf("surface emit failed")
			}
			return nil
		},
	}

	root.AddCommand(validateCmd, astCmd, inspectCmd, renderCmd, emitCmd)
	return root
}

func main() {
	cmd := NewRootCommand(os.Stdout)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
