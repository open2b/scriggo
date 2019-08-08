// run

package main

import "fmt"

func main() {
	a := 1
	b := 20
	fmt.Print(a == b)
	fmt.Print(bool(a == b))
	fmt.Print(bool(a != b))
	fmt.Print(a + b)
	fmt.Print(int(a) + int(b))
}
