// run

package main

import "fmt"

func main() {
	if []int{} == nil {
		fmt.Print("slice is nil")
	} else {
		fmt.Print("slice is not nil")
	}
}
