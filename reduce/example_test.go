package reduce_test

import (
	"fmt"
	"log"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/reduce"
	"gonum.org/v1/gonum/mat"
)

// When a distributed system has both internal retry states and redundant
// multi-zone replicas, combine reduce.Lump (spatial quotienting) and
// reduce.Trace (temporal censoring) to compress the state machine while
// preserving exact transition probabilities.
func Example() {
	// Story: 4 worker pods (a0, a1 in Zone A; b0, b1 in Zone B) process jobs.
	// Within each zone, pods have identical aggregate transition probabilities
	// into Zone A (50%) and Zone B (50%) for Zone A pods, and (30%, 70%) for Zone B pods.
	P := mat.NewDense(4, 4, []float64{
		0.2, 0.3, 0.4, 0.1, // a0 -> ZoneA=0.5, ZoneB=0.5
		0.1, 0.4, 0.2, 0.3, // a1 -> ZoneA=0.5, ZoneB=0.5
		0.3, 0.0, 0.5, 0.2, // b0 -> ZoneA=0.3, ZoneB=0.7
		0.1, 0.2, 0.6, 0.1, // b1 -> ZoneA=0.3, ZoneB=0.7
	})
	k, err := markov.NewKernel(P, []string{"a0", "a1", "b0", "b1"})
	if err != nil {
		log.Fatal(err)
	}

	zones := markov.Partition{{0, 1}, {2, 3}}
	ok, err := reduce.IsStronglyLumpable(k, zones, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	lumped, err := reduce.Lump(k, zones, []string{"ZoneA", "ZoneB"}, nil, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Strongly lumpable by zone: %v\n", ok)
	fmt.Printf("P(ZoneA -> ZoneB): %.2f, P(ZoneB -> ZoneA): %.2f\n", lumped.P.At(0, 1), lumped.P.At(1, 0))
	// Output:
	// Strongly lumpable by zone: true
	// P(ZoneA -> ZoneB): 0.50, P(ZoneB -> ZoneA): 0.30
}

// When a state machine has noisy internal retry or backoff states that you
// do not want to monitor directly, use reduce.Trace and reduce.TraceReward
// to fold them away while preserving exact completion probabilities and
// expected latencies between outer states.
func ExampleTrace() {
	// Story: An RPC request enters at Start (0). On transient errors it enters
	// backoff states Retry1 (1) and Retry2 (2) before reaching Done (3) or Failed (4).
	P := mat.NewDense(5, 5, []float64{
		0.0, 0.3, 0.0, 0.6, 0.1, // Start:  60% Done, 30% Retry1, 10% Failed
		0.0, 0.0, 0.4, 0.4, 0.2, // Retry1: 40% Done, 40% Retry2, 20% Failed
		0.0, 0.0, 0.0, 0.5, 0.5, // Retry2: 50% Done, 50% Failed
		0.0, 0.0, 0.0, 1.0, 0.0, // Done   (absorbing)
		0.0, 0.0, 0.0, 0.0, 1.0, // Failed (absorbing)
	})

	k, err := markov.NewKernel(P, []string{"Start", "Retry1", "Retry2", "Done", "Failed"})
	if err != nil {
		log.Fatal(err)
	}

	// Keep only outer boundary states: {0: Start, 3: Done, 4: Failed}.
	watched := []int{0, 3, 4}
	traced, err := reduce.Trace(k, watched, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	// Fold step latencies (ms) across hidden retries:
	//   Start=10ms, Retry1=50ms, Retry2=150ms.
	stepMs := []float64{10, 50, 150, 0, 0}
	latency, err := reduce.TraceReward(k, stepMs, watched, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Watched states:    %v\n", traced.StateNames)
	fmt.Printf("P(Start -> Done):  %.2f\n", traced.P.At(0, 1))
	fmt.Printf("P(Start -> Fail):  %.2f\n", traced.P.At(0, 2))
	fmt.Printf("Expected latency:  %.1f ms\n", latency[0])
	// Output:
	// Watched states:    [Start Done Failed]
	// P(Start -> Done):  0.78
	// P(Start -> Fail):  0.22
	// Expected latency:  43.0 ms
}

// When worker replicas are supposed to behave symmetrically across availability
// zones, check reduce.LumpabilityResidual to detect silent configuration skew
// before lumping.
func ExampleLumpabilityResidual() {
	// Story: Pod 1 in Zone A has a misconfigured timeout, giving it a 70% transition
	// rate to Zone B while Pod 0 in Zone A only transitions to Zone B 50% of the time.
	// Meanwhile Pods 2 and 3 in Zone B both route 15% to Pod 0 and 15% to Pod 1.
	skewed := mat.NewDense(4, 4, []float64{
		0.20, 0.30, 0.40, 0.10, // Pod 0 (Zone A): 50% to Zone B
		0.10, 0.20, 0.40, 0.30, // Pod 1 (Zone A): 70% to Zone B (20% skew!)
		0.15, 0.15, 0.50, 0.20, // Pod 2 (Zone B): 15% to Pod 0, 15% to Pod 1, 70% to Zone B
		0.15, 0.15, 0.60, 0.10, // Pod 3 (Zone B): 15% to Pod 0, 15% to Pod 1, 70% to Zone B
	})
	k := markov.MustNewKernel(skewed, []string{"a0", "a1", "b0", "b1"})
	zones := markov.Partition{{0, 1}, {2, 3}}

	res, err := reduce.LumpabilityResidual(k, zones)
	if err != nil {
		log.Fatal(err)
	}
	coarsest, err := reduce.CoarsestPartition(k, zones, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Zone lumpability residual: %.2f\n", res)
	fmt.Printf("Refined partition splits skewed pod: %v\n", coarsest)
	// Output:
	// Zone lumpability residual: 0.20
	// Refined partition splits skewed pod: [[0] [1] [2 3]]
}
