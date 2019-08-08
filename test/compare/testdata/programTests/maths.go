// run

package main

import (
	"fmt"
)

func eval(v int) {
	fmt.Print(v, ",")
}

func main() {
	a := 33
	b := 42
	eval(a + b)
	eval(a - b)
	eval(a * b)
	eval(a / b)
	eval(a % b)
	eval(a + b*b)
	eval(a*b + a - b)
	eval(a - b - b + a*b)
	eval(a/3 + b/10)
}
