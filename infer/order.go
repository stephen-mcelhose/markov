package infer

import (
	"fmt"
	"math"

	"github.com/stephen-mcelhose/markov"
)

// MarkovOrderTest performs the Anderson-Goodman / Billingsley log-likelihood
// ratio test (G²) of the null hypothesis H₀ (first-order Markov chain) against
// the alternative H₁ (second-order Markov chain) on a state trajectory:
//
//	G² = 2 Σ_{i,j,k} N[i,j,k] ln( P̂₂[i,j,k] / P̂₁[j,k] )   (0 ln 0 := 0)
//
// where N[i,j,k] counts triples (s_{t-1}=i, s_t=j, s_{t+1}=k),
// P̂₂[i,j,k] = N[i,j,k] / N[i,j,·], and P̂₁[j,k] = N[·,j,k] / N[·,j,·].
//
// Under H₀, G² is asymptotically distributed as χ² with df = n(n-1)² degrees
// of freedom. Returns (chi2, pValue, nil), or an error if len(states) < 3,
// n < 2, or any state index is out of range [0, n).
func MarkovOrderTest(states []int, n int) (chi2, pValue float64, err error) {
	if n < 2 {
		return 0, 0, fmt.Errorf("%w: state count n=%d must be >= 2", markov.ErrNotStochastic, n)
	}
	if len(states) < 3 {
		return 0, 0, fmt.Errorf("%w: sequence length %d must be >= 3", markov.ErrInvalidPartition, len(states))
	}
	if err := validateStateSequenceBounds(states, n); err != nil {
		return 0, 0, err
	}

	counts := countTransitionTriples(states, n)
	g2 := counts.gSquareStatistic()
	df := float64(n * (n - 1) * (n - 1))
	return g2, chiSquareSurvival(g2, df), nil
}

// ── internal helpers: 3-gram contingency tables & G² statistic ────────────────

// tripleContingencyTable holds the 3-gram transition counts N[i,j,k] and
// marginal sums N[i,j,·], N[·,j,k], and N[·,j,·] over an n-state trajectory.
type tripleContingencyTable struct {
	n    int
	nijk []float64
	nij  []float64
	njk  []float64
	nj   []float64
}

// countTransitionTriples tabulates N[i,j,k], N[i,j,·], N[·,j,k], and N[·,j,·]
// across all consecutive triples (s_t, s_{t+1}, s_{t+2}) in states.
func countTransitionTriples(states []int, n int) *tripleContingencyTable {
	tab := &tripleContingencyTable{
		n:    n,
		nijk: make([]float64, n*n*n),
		nij:  make([]float64, n*n),
		njk:  make([]float64, n*n),
		nj:   make([]float64, n),
	}
	for t := 0; t+2 < len(states); t++ {
		i, j, k := states[t], states[t+1], states[t+2]
		tab.nijk[(i*n+j)*n+k]++
		tab.nij[i*n+j]++
		tab.njk[j*n+k]++
		tab.nj[j]++
	}
	return tab
}

// gSquareStatistic computes the log-likelihood ratio statistic:
//
//	G² = 2 Σ_{i,j,k} N[i,j,k] ln( P̂₂[i,j,k] / P̂₁[j,k] )   (with 0 ln 0 := 0)
func (tab *tripleContingencyTable) gSquareStatistic() float64 {
	var g2 float64
	for i := 0; i < tab.n; i++ {
		for j := 0; j < tab.n; j++ {
			g2 += tab.cellGroupGSquare(i, j)
		}
	}
	if g2 < 0 {
		return 0
	}
	return g2
}

// cellGroupGSquare computes 2 Σ_k N[i,j,k] ln( P̂₂[i,j,k] / P̂₁[j,k] ) for a
// fixed history pair (i, j).
func (tab *tripleContingencyTable) cellGroupGSquare(i, j int) float64 {
	n := tab.n
	denomIJ := tab.nij[i*n+j]
	denomJ := tab.nj[j]
	if denomIJ == 0 || denomJ == 0 {
		return 0
	}
	var sum float64
	baseIdx := (i*n + j) * n
	for k := 0; k < n; k++ {
		c := tab.nijk[baseIdx+k]
		p1 := tab.njk[j*n+k] / denomJ
		if c > 0 && p1 > 0 {
			p2 := c / denomIJ
			sum += 2.0 * c * math.Log(p2/p1)
		}
	}
	return sum
}

// ── internal helpers: χ² survival function Q(df/2, x/2) ──────────────────────

// chiSquareSurvival computes P(X ≥ x) for X ~ χ²(df) = Q(df/2, x/2), the
// regularized upper incomplete gamma function.
func chiSquareSurvival(x, df float64) float64 {
	if x <= 0 || df <= 0 {
		return 1.0
	}
	return regGammaUpper(0.5*df, 0.5*x)
}

// regGammaUpper computes Q(a, x) = 1 - P(a, x) using the power series when
// x < a + 1 and Lentz's continued fraction when x ≥ a + 1.
func regGammaUpper(a, x float64) float64 {
	if x <= 0 {
		return 1.0
	}
	if x < a+1.0 {
		return clampProbabilityBounds(1.0 - regGammaLowerSeries(a, x))
	}
	return clampProbabilityBounds(regGammaUpperCF(a, x))
}

// regGammaLowerSeries evaluates the regularized lower incomplete gamma P(a, x)
// via its Taylor series P(a, x) = e⁻ˣ xᵃ / Γ(a) Σ_{k=0}^∞ xᵏ / (a(a+1)…(a+k)).
func regGammaLowerSeries(a, x float64) float64 {
	lg, _ := math.Lgamma(a)
	ap := a
	sum := 1.0 / a
	del := sum
	for iter := 0; iter < 500; iter++ {
		ap += 1.0
		del *= x / ap
		sum += del
		if math.Abs(del) < math.Abs(sum)*1e-15 {
			break
		}
	}
	return sum * math.Exp(-x+a*math.Log(x)-lg)
}

// regGammaUpperCF evaluates the regularized upper incomplete gamma Q(a, x)
// via Legendre's continued fraction and modified Lentz's method.
func regGammaUpperCF(a, x float64) float64 {
	const tiny = 1e-30
	lg, _ := math.Lgamma(a)
	b := x + 1.0 - a
	c := 1.0 / tiny
	d := 1.0 / b
	h := d
	for i := 1; i <= 500; i++ {
		fi := float64(i)
		an := -fi * (fi - a)
		b += 2.0
		d = clampTinyDenominator(an*d+b, tiny)
		c = clampTinyDenominator(b+an/c, tiny)
		d = 1.0 / d
		del := d * c
		h *= del
		if math.Abs(del-1.0) < 1e-15 {
			break
		}
	}
	return math.Exp(-x+a*math.Log(x)-lg) * h
}

// clampTinyDenominator prevents division by zero in Lentz's continued fraction.
func clampTinyDenominator(v, tiny float64) float64 {
	if math.Abs(v) < tiny {
		return tiny
	}
	return v
}

// clampProbabilityBounds clamps a p-value onto [0, 1].
func clampProbabilityBounds(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
