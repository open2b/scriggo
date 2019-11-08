// skip

package main

type Int int
type SumFunc func(a, b Int) Int

func main() {
	var _ func(A Int) SumFunc = func(A Int) SumFunc { return nil }
}
