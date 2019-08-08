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
		g := func() {
			fmt.Print(", g:", A)
			A = 30
			fmt.Print(", g:", A)
		}
		fmt.Print(", f:", A)
		g()
		fmt.Print(", f:", A)
	}
	fmt.Print(", main:", A)
	f()
	fmt.Print(", main:", A)
}
