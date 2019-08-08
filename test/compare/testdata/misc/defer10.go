// skip : cannot pass the argument

// run

package main

import (
	"fmt"
)

func main() {
	defer func(s []int) {
		fmt.Println(s)
	}([]int{1, 2, 3})
}
