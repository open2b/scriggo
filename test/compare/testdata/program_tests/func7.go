// run

package main

import "fmt"

func main() {
	a := 41
	fmt.Print(a)
	inc := func(a int) int {
		fmt.Print("inc:", a)
		b := a
		b++
		fmt.Print("inc:", b)
		return b
	}
	a = inc(a)
	fmt.Print(a)
	return
}
