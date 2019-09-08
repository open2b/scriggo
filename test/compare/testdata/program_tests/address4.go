// run

package main

import "fmt"

type Int2 = Int1
type Int1 = int

func main() {
	var i1 Int1 = Int1(10)
	var i2 Int2 = Int2(30)
	fmt.Print(i1, i2)
}
