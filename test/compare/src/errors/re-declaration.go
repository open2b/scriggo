// errorcheck

package main

import (
	"fmt"
)

func main() {
	x := 1

	x := 2 // ERROR `no new variables on left side of :=`

	fmt.Println(x)
}
