// run

package main

import (
	"fmt"
	"strings"
	"unicode"
)

func main() {
	f := func(c rune) bool {
		return unicode.Is(unicode.Han, c)
	}
	fmt.Print(strings.IndexFunc("Hello, 世界\n", f))
	fmt.Print(strings.IndexFunc("Hello, world\n", f))
}
