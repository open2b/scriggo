// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print(3 + 3i)
	c1 := 3 + 3 + 19 - 3i - 120i
	fmt.Print(c1)
	fmt.Print(-c1)
	fmt.Print(c1 * c1)
	c2 := 321.43 + 3i - 32.129i
	fmt.Print(c2)
	fmt.Print(-c2)
	fmt.Print(c2 * c2)
	fmt.Print(c1 * c2)
}
