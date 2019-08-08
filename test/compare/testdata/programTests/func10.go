// run

package main

import "fmt"

func main() {
	f := func() int {
		a := 20
		fmt.Print("f called")
		return a
	}
	_ = f
}
