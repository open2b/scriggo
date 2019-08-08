// run

package main

import (
	"fmt"
)

func main() {
	pi := new(int)
	fmt.Printf("pi: %T,", pi)
	pi2 := new(int)
	pi2 = pi
	equal := pi == pi2
	fmt.Print("equal?", equal)
}
