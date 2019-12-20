// run

package main

import "fmt"

func main() {
	i := interface{}(2)
	switch v := i.(type) {
	case int:
		fmt.Print(v)
	default:
		_ = v
		panic("")
	}
}
