// run

package main

import "fmt"

func main() {
	a := float32(16.0)
	b := float32(16.0)

	_ = int(a) // w/o this everything works as expected

	if a != b {
		panic(fmt.Sprintf("a (%v) != b (%v)", a, b))
	}
}
