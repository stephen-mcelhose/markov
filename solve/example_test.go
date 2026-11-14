package solve_test

import (
	"fmt"
	"log"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/solve"
	"gonum.org/v1/gonum/mat"
)

// Don't use naive averages on cyclic failure graphs; use solve.Stationary for
// long-run availability, solve.MFPTToTarget for exact mean recovery steps, and
// solve.MFPTDelta to find the single transition improvement with the highest ROI.
func Example() {
	// Story: A microservice cycles between 4 operational states:
	//   0: Healthy
	//   1: Degraded     (elevated latency)
	//   2: CircuitOpen  (shedding traffic)
	//   3: Rebooting    (container restart)
	P := mat.NewDense(4, 4, []float64{
		0.90, 0.10, 0.00, 0.00, // Healthy     -> 90% stay, 10% Degraded
		0.40, 0.30, 0.30, 0.00, // Degraded    -> 40% recover, 30% stay, 30% CircuitOpen
		0.00, 0.00, 0.50, 0.50, // CircuitOpen -> 50% stay open, 50% Rebooting
		0.80, 0.20, 0.00, 0.00, // Rebooting   -> 80% Healthy, 20% Degraded
	})
	k, err := markov.NewKernel(P, []string{"Healthy", "Degraded", "CircuitOpen", "Rebooting"})
	if err != nil {
		log.Fatal(err)
	}

	// 1. Long-run steady-state availability via GTH elimination:
	pi, err := solve.Stationary(k, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Mean steps to return to Healthy (target = 0) from every state:
	recoverySteps, err := solve.MFPTToTarget(k, 0, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Compare two engineering fixes from CircuitOpen (state 2) toward Healthy (0):
	//    Option A: Fast-path 50% of CircuitOpen directly to Healthy (2 -> 0, delta=0.5).
	//    Option B: Fast-path 50% of CircuitOpen to Rebooting       (2 -> 3, delta=0.5).
	roiDirect, err := solve.MFPTDelta(k, 2, 0, 0, 0.50, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	roiReboot, err := solve.MFPTDelta(k, 2, 3, 0, 0.50, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Steady-state Healthy availability: %.1f%%\n", 100*pi[0])
	fmt.Printf("Mean recovery steps from CircuitOpen -> Healthy: %.2f steps\n", recoverySteps[2])
	fmt.Printf("Recovery delta (Option A direct probe):  %.2f steps\n", roiDirect)
	fmt.Printf("Recovery delta (Option B faster reboot): %.2f steps\n", roiReboot)
	// Output:
	// Steady-state Healthy availability: 77.1%
	// Mean recovery steps from CircuitOpen -> Healthy: 3.59 steps
	// Recovery delta (Option A direct probe):  -1.78 steps
	// Recovery delta (Option B faster reboot): -0.73 steps
}

// When analyzing a state graph that may contain absorbing failure traps or
// periodic cycles, use solve.ClassifyStates to decompose SCCs, recurrent
// classes, and transient states before running ergodic solvers.
func ExampleClassifyStates() {
	// State 0 is a transient startup state that enters a recurrent {1, 2} loop.
	P := mat.NewDense(3, 3, []float64{
		0.2, 0.8, 0.0,
		0.0, 0.5, 0.5,
		0.0, 0.5, 0.5,
	})
	k := markov.MustNewKernel(P, []string{"Init", "ActiveA", "ActiveB"})
	classes, err := solve.ClassifyStates(k, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Transient: %v, Recurrent: %v\n", classes.Transient, classes.Recurrent)
	// Output:
	// Transient: [0], Recurrent: [[1 2]]
}

// When comparing a baseline deployment kernel against a canary kernel, use
// solve.EntropyRate and solve.KLDivergenceRate to quantify unpredictability and
// behavioral drift in bits per transition step.
func ExampleKLDivergenceRate() {
	baseline := markov.MustNewKernel(mat.NewDense(2, 2, []float64{
		0.9, 0.1,
		0.5, 0.5,
	}), []string{"OK", "Err"})
	canary := markov.MustNewKernel(mat.NewDense(2, 2, []float64{
		0.7, 0.3,
		0.5, 0.5,
	}), []string{"OK", "Err"})

	hBase, err := solve.EntropyRate(baseline, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	drift, err := solve.KLDivergenceRate(canary, baseline, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Baseline entropy rate: %.3f bits/step, Canary KL drift: %.3f bits/step\n", hBase, drift)
	// Output:
	// Baseline entropy rate: 0.557 bits/step, Canary KL drift: 0.139 bits/step
}
