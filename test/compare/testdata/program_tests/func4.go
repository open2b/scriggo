// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("start,")
	func() {
		fmt.Print("f,")
	}()
	fmt.Print("end")
}
