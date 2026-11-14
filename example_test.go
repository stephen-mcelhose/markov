package markov_test

import (
	"fmt"
	"log"

	"github.com/stephen-mcelhose/markov"
	"gonum.org/v1/gonum/mat"
)

// When raw telemetry gives unnormalized transition counts or per-pod traffic
// shares across availability zones, normalize into a validated markov.Kernel
// and use Distribution.Restrict (subset conditioning) and Distribution.Project
// (zone rollups) instead of manual slice loops.
func Example() {
	// Story: A load balancer routes requests across 4 pods in two availability
	// zones: us-east-1a (pods 0, 1) and us-east-1b (pods 2, 3).
	// Raw transition counts from a 1-minute window are unnormalized.
	rawCounts := mat.NewDense(4, 4, []float64{
		60, 20, 10, 10,
		15, 65, 10, 10,
		10, 10, 50, 30,
		10, 10, 25, 55,
	})
	if err := markov.NormalizeRows(rawCounts, markov.DefaultTolerance); err != nil {
		log.Fatal(err)
	}

	k, err := markov.NewKernel(rawCounts, []string{"east1a-p0", "east1a-p1", "east1b-p2", "east1b-p3"})
	if err != nil {
		log.Fatal(err)
	}

	// Steady traffic share across the 4 pods:
	podShare := markov.MustNewDistribution([]float64{0.30, 0.20, 0.25, 0.25}, markov.DefaultTolerance)

	// Roll up pod traffic into availability zones {{0, 1}, {2, 3}}.
	zones := markov.Partition{{0, 1}, {2, 3}}
	zoneShare, err := podShare.Project(zones, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	// Condition traffic share onto only the us-east-1a pods {0, 1} during a zone drill.
	east1aOnly, err := podShare.Restrict([]int{0, 1}, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("States: %d (%s -> %s = %.2f)\n", k.NumStates(), k.StateNames[0], k.StateNames[1], k.P.At(0, 1))
	fmt.Printf("Zone traffic share [us-east-1a, us-east-1b]: [%.2f, %.2f]\n", zoneShare[0], zoneShare[1])
	fmt.Printf("Within us-east-1a conditional share [p0, p1]: [%.2f, %.2f]\n", east1aOnly[0], east1aOnly[1])
	// Output:
	// States: 4 (east1a-p0 -> east1a-p1 = 0.20)
	// Zone traffic share [us-east-1a, us-east-1b]: [0.50, 0.50]
	// Within us-east-1a conditional share [p0, p1]: [0.60, 0.40]
}

// When wrapping a Gonum matrix as a Markov chain, use markov.NewKernel to
// enforce row-stochasticity and attach human-readable state names up front.
func ExampleNewKernel() {
	P := mat.NewDense(2, 2, []float64{
		0.85, 0.15,
		0.40, 0.60,
	})
	k, err := markov.NewKernel(P, []string{"Healthy", "Degraded"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("States: %v, P(Healthy->Degraded)=%.2f\n", k.StateNames, k.P.At(0, 1))
	// Output:
	// States: [Healthy Degraded], P(Healthy->Degraded)=0.15
}

// When inspecting conditional behavior inside an active subset of states (such
// as non-drained pods), use Distribution.Restrict to slice and renormalize the
// probability vector in one step.
func ExampleDistribution_Restrict() {
	pi := markov.MustNewDistribution([]float64{0.45, 0.30, 0.25}, markov.DefaultTolerance)
	active, err := pi.Restrict([]int{0, 1}, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Active subset distribution: [%.2f, %.2f]\n", active[0], active[1])
	// Output:
	// Active subset distribution: [0.60, 0.40]
}

// When rolling up micro-state probabilities into macro-groups (such as regions
// or service tiers), use Distribution.Project with a markov.Partition.
func ExampleDistribution_Project() {
	pi := markov.MustNewDistribution([]float64{0.15, 0.35, 0.20, 0.30}, markov.DefaultTolerance)
	tiers := markov.Partition{{0, 1}, {2, 3}}
	macro, err := pi.Project(tiers, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Macro-tier probabilities: [%.2f, %.2f]\n", macro[0], macro[1])
	// Output:
	// Macro-tier probabilities: [0.50, 0.50]
}

// When lumping a state space into macro-blocks, use Partition.DistributorMatrix
// and Partition.IndicatorMatrix to construct the Kemeny-Snell projection pair
// (U, V) satisfying U*V = I.
func ExamplePartition_DistributorMatrix() {
	p := markov.Partition{{0, 1}, {2, 3}}
	V, err := p.IndicatorMatrix(4)
	if err != nil {
		log.Fatal(err)
	}
	U, err := p.DistributorMatrix(4, nil, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	var UV mat.Dense
	UV.Mul(U, V)
	fmt.Printf("U*V diagonal: [%.1f, %.1f]\n", UV.At(0, 0), UV.At(1, 1))
	// Output:
	// U*V diagonal: [1.0, 1.0]
}
