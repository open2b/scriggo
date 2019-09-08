// run

package main

import "fmt"

func main() {
	f := func(a int) {
		fmt.Print("f called")
		b := a + 20
		_ = b
	}
	_ = f
}
