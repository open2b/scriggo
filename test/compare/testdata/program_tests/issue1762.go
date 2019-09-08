// run

package main

import (
	"fmt"
	"unicode"
)

func main() {
	f := func() {
		fmt.Println(unicode.Cc)
	}
	f()
}
