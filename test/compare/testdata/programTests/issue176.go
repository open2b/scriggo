// run

package main

import (
	"unicode"
)

func main() {
	f := func() {
		_ = unicode.Cc
	}
	_ = f
}
