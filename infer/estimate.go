package infer

import (
	"fmt"
	"math"
	"slices"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// Estimator accumulates observed state transitions with a symmetric Dirichlet
// prior α ≥ 0 and optional exponential moving average (EMA) half-life h ≥ 0.
type Estimator struct {
	numStates   int
	stateNames  []string
	priorAlpha  float64
	halfLife    float64
	decayFactor float64
	counts      *mat.Dense
}

// NewEstimator constructs an Estimator for n states with prior α ≥ 0 and
// half-life h ≥ 0:
//
//	C⁽⁰⁾[i, j] = 0
//	γ = 1         if h = 0
//	γ = 2^(-1/h)  if h > 0
//
// Returns an error if n < 1, prior < 0, halflife < 0, or len(names) ∉ {0, n}.
func NewEstimator(n int, names []string, prior, halflife float64) (*Estimator, error) {
	if err := validateEstimatorParams(n, names, prior, halflife); err != nil {
		return nil, err
	}
	return &Estimator{
		numStates:   n,
		stateNames:  slices.Clone(names),
		priorAlpha:  prior,
		halfLife:    halflife,
		decayFactor: decayFactorFromHalfLife(halflife),
		counts:      mat.NewDense(n, n, nil),
	}, nil
}

// Observe records a single transition from → to:
//
//	C⁽ᵗ⁾[i, j] = γ·C⁽ᵗ⁻¹⁾[i, j] + 1_{(i = from, j = to)}
//
// Returns an error if e is nil or from/to are out of range [0, n).
func (e *Estimator) Observe(from, to int) error {
	if e == nil || e.counts == nil {
		return fmt.Errorf("%w: nil estimator", markov.ErrNotStochastic)
	}
	if from < 0 || from >= e.numStates || to < 0 || to >= e.numStates {
		return fmt.Errorf("%w: transition (%d -> %d) out of range [0,%d)", markov.ErrInvalidPartition, from, to, e.numStates)
	}
	if e.decayFactor != 1.0 {
		e.counts.Scale(e.decayFactor, e.counts)
	}
	e.counts.Set(from, to, e.counts.At(from, to)+1.0)
	return nil
}

// ObserveSequence records consecutive transitions s[t] → s[t+1] for t ∈ {0, …, len(states)-2}.
// If len(states) < 2, it validates any present state index and returns nil without mutating counts.
// Returns an error if e is nil or any state index is out of range [0, n).
func (e *Estimator) ObserveSequence(states []int) error {
	if e == nil || e.counts == nil {
		return fmt.Errorf("%w: nil estimator", markov.ErrNotStochastic)
	}
	if err := validateStateSequenceBounds(states, e.numStates); err != nil {
		return err
	}
	for t := 0; t+1 < len(states); t++ {
		if err := e.Observe(states[t], states[t+1]); err != nil {
			return err
		}
	}
	return nil
}

// Kernel computes the maximum a posteriori / posterior predictive transition
// kernel from accumulated counts and prior α:
//
//	P̂[i, j] = (C[i, j] + α) / Σ_r (C[i, r] + α)
//
// Returns [markov.ErrNotStochastic] if e is nil, tol < 0, or any row has total
// mass Σ_r (C[i, r] + α) ≤ tol.
func (e *Estimator) Kernel(tol float64) (*markov.Kernel, error) {
	if e == nil || e.counts == nil {
		return nil, fmt.Errorf("%w: nil estimator", markov.ErrNotStochastic)
	}
	if tol < 0 || math.IsNaN(tol) {
		return nil, fmt.Errorf("%w: invalid tolerance %g", markov.ErrNotStochastic, tol)
	}
	P := mat.NewDense(e.numStates, e.numStates, nil)
	for i := 0; i < e.numStates; i++ {
		if err := normalizePosteriorRow(P.RawRowView(i), e.counts.RawRowView(i), e.priorAlpha, i, tol); err != nil {
			return nil, err
		}
	}
	return markov.NewKernelWithTol(P, e.stateNames, math.Max(tol, markov.DefaultTolerance))
}

// FromTransitions estimates a Kernel from interleaved transition pairs
// [from₀, to₀, from₁, to₁, …] with Dirichlet prior α and no decay (h = 0).
//
// Returns an error if len(pairs) is odd, any state index is out of range [0, n),
// or any row has zero total mass.
func FromTransitions(pairs []int, n int, prior float64, tol float64) (*markov.Kernel, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("%w: pairs slice length %d must be even", markov.ErrInvalidPartition, len(pairs))
	}
	est, err := NewEstimator(n, nil, prior, 0)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(pairs); i += 2 {
		if err := est.Observe(pairs[i], pairs[i+1]); err != nil {
			return nil, err
		}
	}
	return est.Kernel(tol)
}

// FromSequence estimates a Kernel from a sequential state trajectory
// [s₀, s₁, …, s_{T-1}] with Dirichlet prior α and EMA half-life h.
//
// Returns an error if len(states) < 2, any state index is out of range [0, n),
// parameters are invalid, or any row has zero total mass.
func FromSequence(states []int, n int, prior, halflife, tol float64) (*markov.Kernel, error) {
	if len(states) < 2 {
		return nil, fmt.Errorf("%w: sequence length %d must be >= 2", markov.ErrInvalidPartition, len(states))
	}
	est, err := NewEstimator(n, nil, prior, halflife)
	if err != nil {
		return nil, err
	}
	if err := est.ObserveSequence(states); err != nil {
		return nil, err
	}
	return est.Kernel(tol)
}

// ── internal helpers ──────────────────────────────────────────────────────────

// validateEstimatorParams checks constructor preconditions for NewEstimator.
func validateEstimatorParams(n int, names []string, prior, halflife float64) error {
	if n < 1 {
		return fmt.Errorf("%w: state count n=%d must be >= 1", markov.ErrNotStochastic, n)
	}
	if prior < 0 || math.IsNaN(prior) || math.IsInf(prior, 0) {
		return fmt.Errorf("%w: invalid prior %g (must be >= 0)", markov.ErrInvalidDistribution, prior)
	}
	if halflife < 0 || math.IsNaN(halflife) {
		return fmt.Errorf("%w: invalid halflife %g (must be >= 0)", markov.ErrInvalidDistribution, halflife)
	}
	if len(names) > 0 && len(names) != n {
		return fmt.Errorf("%w: %d names for %d states", markov.ErrNotStochastic, len(names), n)
	}
	return nil
}

// decayFactorFromHalfLife computes the per-event exponential decay multiplier
// γ = 2^(-1/h) when h > 0 (or γ = 1 when h = 0 or +Inf).
func decayFactorFromHalfLife(halflife float64) float64 {
	if halflife > 0 && !math.IsInf(halflife, 1) {
		return math.Pow(2.0, -1.0/halflife)
	}
	return 1.0
}

// validateStateSequenceBounds checks that every state s in states lies in [0, n).
func validateStateSequenceBounds(states []int, n int) error {
	for _, s := range states {
		if s < 0 || s >= n {
			return fmt.Errorf("%w: state index %d out of range [0,%d)", markov.ErrInvalidPartition, s, n)
		}
	}
	return nil
}

// normalizePosteriorRow computes the posterior predictive row probabilities
// dst[j] = (counts[j] + prior) / Σ_r (counts[r] + prior) for state rowIdx.
func normalizePosteriorRow(dst, counts []float64, prior float64, rowIdx int, tol float64) error {
	var rowSum float64
	for j, c := range counts {
		v := c + prior
		dst[j] = v
		rowSum += v
	}
	if rowSum <= tol {
		return fmt.Errorf("%w: state %d has zero total count and zero prior", markov.ErrNotStochastic, rowIdx)
	}
	for j := range dst {
		dst[j] /= rowSum
	}
	return nil
}
