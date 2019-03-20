//+build ignore

package main

import (
	"fmt"
)

func main() {
	x := 1

	// TODO (Gianluca): Go shows error at ":=", while Scrigo points to "x"
	// x := 2

	fmt.Println(x)
}
