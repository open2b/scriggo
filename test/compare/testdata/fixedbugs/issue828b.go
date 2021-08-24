// run

package main

import "fmt"

var x = "package-level"

func main() {
	var x string
	x = "scope"
	_ = x
	fmt.Println(x)
}
