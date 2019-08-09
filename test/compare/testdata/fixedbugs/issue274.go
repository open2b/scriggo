// run

package main

import "fmt"

func main() {
	type t = struct{ A int }
	fmt.Print(t{})
}
