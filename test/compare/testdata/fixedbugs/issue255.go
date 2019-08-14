// run

package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	test1()
	test2()
}

func test1() {
	var s string
	s = "abc"[0:1]
	fmt.Println(s)
}

func test2() {
	s := "\000\123\x00\xca\xFE\u0123\ubabe\U0000babe\U0010FFFFx"
	fmt.Println(utf8.DecodeRuneInString(s[0:2]))
}
