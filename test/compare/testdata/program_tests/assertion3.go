// run

package main

import "fmt"

func main() {
	i := interface{}(5)
	a := 7 + i.(int)
	fmt.Print(i, ", ", a)
	return
}
