// run

package main

import "fmt"

func main() {

	i := 0
	e := 0
	for i, e = range []int{3, 4, 5} {
		fmt.Print(i, e, ",")
	}
}
