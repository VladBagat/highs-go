package main

import (
	"fmt"
	"os"

	highs "github.com/VladBagat/highs-go"
)

func main() {
	model := highs.Model{Sense: highs.Maximize, Offset: 0}

	m := model.AddVariable(0, 0, 4)
	f := model.AddVariable(1, 0, 3)

	model.AddConstraint(0, 0, highs.Term{Variable: m, Coefficient: 1}, highs.Term{Variable: f, Coefficient: -2})

	r, err := model.Solve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("status=%s\n", r.Status)
	if r.Status == highs.Optimal {
		fmt.Printf("objective=%.6g x=%.6g y=%.6g\n", r.Objective, r.Values[m], r.Values[f])
	}
}
