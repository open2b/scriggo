// run

package main

import "fmt"

func main() {

	for i, v := range []int{4, 5, 6} {
		fmt.Print(i, v, ",")
	}

}
