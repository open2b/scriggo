// run

package main

import "fmt"

type T = int

var v1 T = 10

func main() {
	fmt.Printf("%v-%T", v1, v1)
	var v2 T = 20
	fmt.Printf("%v-%T", v2, v2)
	type T = string
	var v3 T = "hey"
	fmt.Printf("%v-%T", v3, v3)
}
