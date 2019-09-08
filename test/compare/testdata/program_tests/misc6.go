// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("start, ")
	for _, v := range []int{1, 2, 3} {
		fmt.Print("v: ", v, ", ")
		if v == 2 {
			fmt.Print("break, ")
			break
		}
		fmt.Print("no break, ")
	}
	fmt.Print("end")
}
