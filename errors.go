package markov

import "errors"

// ErrUnimplemented is returned when an API stub has not yet been implemented.
var ErrUnimplemented = errors.New("markov: unimplemented")

// ErrNotStochastic is returned when a matrix fails row-stochasticity validation:
// some entry is below -tol, or some row sum deviates from 1 by more than tol.
var ErrNotStochastic = errors.New("markov: matrix is not row-stochastic")

// ErrInvalidDistribution is returned when a probability vector fails validation:
// some entry is negative, or the entries do not sum to 1 within tolerance.
var ErrInvalidDistribution = errors.New("markov: invalid probability distribution")

// ErrInvalidPartition is returned when a partition fails validation:
// it is empty, contains an empty block, is not disjoint, or does not cover
// all n states.
var ErrInvalidPartition = errors.New("markov: invalid partition")

// ErrSingularOperator is returned when a linear system arising from a
// reduction operator (e.g. Trace, Stationary) is singular and cannot be solved.
var ErrSingularOperator = errors.New("markov: singular operator")
