// run

package main

import (
	"fmt"
)

func g1() (_ string, _ int) {
	return "hello", 2
}

func g2() (_ []int, _ int) {
	return
}

func main() {
	{
		x, y := g1()
		fmt.Println("x:", x, "y:", y)
	}
	{
		x, y := g2()
		fmt.Println("x:", x, "y:", y)
	}
}
