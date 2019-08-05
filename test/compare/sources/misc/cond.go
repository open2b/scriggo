// run

package main

import "fmt"

func main() {

	// Condition is a boolean constant.
	if true {
		fmt.Print("true")
	} else {
		panic("error")
	}

	if false {
		panic("error")
	} else {
		fmt.Print("false")
	}

	if 43 == 43 {
		fmt.Print("43 == 43")
	} else {
		panic("error")
	}

	if !true == (56 > 100) {
		fmt.Print("!true == 56 > 100")
	} else {
		panic("error")
	}
}
