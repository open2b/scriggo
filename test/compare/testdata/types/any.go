// run

package main

import "fmt"

var a any
var b interface{}

func main() {
	_ = &a == &b
	fmt.Printf("a: %T\nb: %T\n", &a, &b)
}
