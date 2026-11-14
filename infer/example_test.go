package infer_test

import (
	"fmt"
	"log"

	"github.com/stephen-mcelhose/markov"
	"github.com/stephen-mcelhose/markov/infer"
)

// When streaming event logs might have non-stationary regime shifts or hidden
// memory, use infer.NewEstimator with an EMA half-life to track live transition
// probabilities and infer.MarkovOrderTest (G²) to check if your state labels
// are missing second-order history.
func Example() {
	// Story: A health monitor observes a stream of service states (0=OK, 1=Alert).
	// In the stream below, Alert (1) is always followed by another Alert (1) if
	// it just entered Alert from OK (0->1->1), and recovers to OK after two
	// consecutive Alerts (1->1->0): [0, 0, 1, 1, 0, 0, 1, 1, ...].
	// A first-order model on {OK, Alert} misses this 2-step duration memory!
	var stream []int
	for i := 0; i < 15; i++ {
		stream = append(stream, 0, 0, 1, 1)
	}

	// 1. Fit a first-order kernel with Laplace smoothing (prior = 1.0):
	k, err := infer.FromSequence(stream, 2, 1.0, 0, markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Test whether the {OK, Alert} labels violate the 1st-order Markov property:
	g2, pVal, err := infer.MarkovOrderTest(stream, 2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("First-order P(Alert -> Alert): %.2f\n", k.P.At(1, 1))
	fmt.Printf("Order test G² = %.1f, p < 0.001: %v (hidden 2nd-order memory detected)\n", g2, pVal < 0.001)
	// Output:
	// First-order P(Alert -> Alert): 0.52
	// Order test G² = 80.3, p < 0.001: true (hidden 2nd-order memory detected)
}

// When estimating transition probabilities from an online event stream where
// recent behavior matters more than old history, use infer.NewEstimator with a
// positive EMA half-life h and Dirichlet prior α.
func ExampleNewEstimator() {
	est, err := infer.NewEstimator(2, []string{"Idle", "Busy"}, 0.5, 4.0)
	if err != nil {
		log.Fatal(err)
	}
	if err := est.ObserveSequence([]int{0, 1, 1, 1, 0, 1}); err != nil {
		log.Fatal(err)
	}
	k, err := est.Kernel(markov.DefaultTolerance)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("P(Idle -> Busy): %.2f, P(Busy -> Busy): %.2f\n", k.P.At(0, 1), k.P.At(1, 1))
	// Output:
	// P(Idle -> Busy): 0.80, P(Busy -> Busy): 0.57
}
