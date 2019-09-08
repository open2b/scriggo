// run

package main

import "fmt"

func main() {
	if []int(nil) == nil {
		fmt.Print("slice is nil")
	} else {
		fmt.Print("slice is not nil")
	}
}
