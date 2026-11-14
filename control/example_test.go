package control_test

import (
	"fmt"
	"log"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/control"
	"gonum.org/v1/gonum/mat"
)

// When you have both state transition probabilities and operating rewards/costs,
// wrap them in control.NewMDP and let control.PolicyIteration synthesize the
// provably optimal control policy.
func Example() {
	// Story: An ingestion queue has 3 load states: Low (0), High (1), Overload (2).
	// The controller chooses between 2 actions:
	//   Action 0 ("NormalRate"): High throughput reward in Low/High, but risks
	//                            staying stuck in Overload with heavy SLA penalty (-20).
	//   Action 1 ("Throttle"):   Lower immediate throughput reward (+4), but
	//                            drains Overload back to Low (70%) and High (30%).
	pNormal := markov.MustNewKernel(mat.NewDense(3, 3, []float64{
		0.70, 0.30, 0.00,
		0.20, 0.50, 0.30,
		0.00, 0.20, 0.80,
	}), []string{"Low", "High", "Overload"})

	pThrottle := markov.MustNewKernel(mat.NewDense(3, 3, []float64{
		0.90, 0.10, 0.00,
		0.60, 0.40, 0.00,
		0.70, 0.30, 0.00,
	}), []string{"Low", "High", "Overload"})

	// Rewards R(s, a): rows = [Low, High, Overload], cols = [NormalRate, Throttle].
	rewards := mat.NewDense(3, 2, []float64{
		10.0, 4.0, // Low:      NormalRate=+10, Throttle=+4
		8.0, 4.0, // High:     NormalRate=+8,  Throttle=+4
		-20.0, 2.0, // Overload: NormalRate=-20, Throttle=+2 (shed load)
	})

	mdp, err := control.NewMDP(
		[]string{"NormalRate", "Throttle"},
		[]*markov.Kernel{pNormal, pThrottle},
		rewards,
		0.90,
		markov.DefaultTolerance,
	)
	if err != nil {
		log.Fatal(err)
	}

	policy, values, err := control.PolicyIteration(mdp, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	for s, stateName := range pNormal.StateNames {
		fmt.Printf("State %-8s -> Action: %-10s (V* = %.1f)\n", stateName, mdp.Actions[policy[s]], values[s])
	}
	// Output:
	// State Low      -> Action: NormalRate (V* = 86.3)
	// State High     -> Action: NormalRate (V* = 81.2)
	// State Overload -> Action: Throttle   (V* = 78.3)
}

// When an MDP has duplicated states across symmetric workers or shards with
// identical rewards and transition profiles, use control.MDPBisimulation to
// find the coarsest state partition that preserves the optimal value function V*.
func ExampleMDPBisimulation() {
	// 4 states where {0, 1} are symmetric idle shards and {2, 3} are symmetric busy shards.
	p0 := markov.MustNewKernel(mat.NewDense(4, 4, []float64{
		0.4, 0.4, 0.1, 0.1,
		0.3, 0.5, 0.2, 0.0,
		0.1, 0.1, 0.4, 0.4,
		0.0, 0.2, 0.5, 0.3,
	}), nil)
	rewards := mat.NewDense(4, 1, []float64{10, 10, 2, 2})

	mdp, err := control.NewMDP([]string{"Serve"}, []*markov.Kernel{p0}, rewards, 0.9, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	blocks, err := control.MDPBisimulation(mdp, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Bisimulation blocks: %v\n", blocks)
	// Output:
	// Bisimulation blocks: [[0 1] [2 3]]
}
