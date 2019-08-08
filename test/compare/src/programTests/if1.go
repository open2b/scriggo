// run

package main

import (
	"fmt"
)

func main() {
	f := func() {
		if true == false {
			fmt.Print("paradox")
		} else {
			fmt.Print("ok")
		}
	}
	f()
}
