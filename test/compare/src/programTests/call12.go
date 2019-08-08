// run

package main

import (
	"fmt"
)

func main() {
	func() {
		func() {
			fmt.Print("called")
		}()
	}()
}
