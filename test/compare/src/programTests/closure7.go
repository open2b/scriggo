// run

package main

import (
	"fmt"
)

func main() {
	A := 5
	fmt.Print("main:", A)
	f := func() {
		fmt.Print(", f:", A)
		A = 20
		fmt.Print(", f:", A)
	}
	fmt.Print(", main:", A)
	f()
	fmt.Print(", main:", A)
}
