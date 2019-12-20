// run

package main

import "fmt"

func main() {
	x := interface{}(int(1))
	switch xx := x.(type) {
	default:
		fmt.Printf("default (%T)", xx)
	case uint64:
		_ = xx
		fmt.Print("unsigned64")
	}
}
