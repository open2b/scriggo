// run

package main

import (
	"fmt"
)

func main() {

	// Constants.
	var i interface{} = -4
	fmt.Print(i)
	fmt.Print(-329)
	a := -43
	fmt.Print(a)

	// Non constants.
	var i2 = 4
	var interf interface{} = -i2
	fmt.Print(interf)
	var b = 329
	fmt.Print(-b)
}
