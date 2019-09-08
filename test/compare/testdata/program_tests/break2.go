// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("switch,")
	switch {
	case true:
		fmt.Print("case true,")
		break
		fmt.Print("???")
	default:
		fmt.Print("???")
	}
	fmt.Print("done")
}
