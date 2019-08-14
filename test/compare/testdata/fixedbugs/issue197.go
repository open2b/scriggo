// run

package main

import (
	"fmt"
)

func main() {
	A := 10
	var (
		ptrA  = &A
		ptrA2 = &(*(ptrA))
	)
	fmt.Println(ptrA == ptrA2)
}
