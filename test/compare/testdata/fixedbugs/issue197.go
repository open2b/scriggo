// run

package main

import (
	"fmt"
)

var a = 4382

func main() {
	{
		pa1 := &a
		pa2 := &a
		fmt.Println(pa1 == pa2)
	}
	{
		A := 10
		var (
			ptrA  = &A
			ptrA2 = &(*(ptrA))
		)
		fmt.Println(ptrA == ptrA2)
	}
}
