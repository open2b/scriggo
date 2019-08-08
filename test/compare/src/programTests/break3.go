// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("switch,")
	switch interface{}("hey").(type) {
	case string:
		fmt.Print("case string,")
		break
		fmt.Print("???")
	default:
		fmt.Print("???")
	}
	fmt.Print("done")
}
