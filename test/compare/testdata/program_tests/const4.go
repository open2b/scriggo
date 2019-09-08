// run

package main

import (
	"fmt"
	"math"
	"time"
)

func main() {
	phi := math.Phi
	fmt.Println("phi:", phi)

	ansic := time.ANSIC
	fmt.Println("ansic:", ansic)

	nanosecond := time.Nanosecond
	fmt.Println("nanosecond:", nanosecond)
}
