// run

package main

import (
	"fmt"
)

func main() {
	t := true
	f := false
	fmt.Print(t && f)
	fmt.Print(t && f || f && t)
	fmt.Print(t || f)
}
