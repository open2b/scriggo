package main

// The situation is the following: this package ('main') imports both 'a' and
// 'b', that both import 'c' (which has an init function). This test checks that
// the init function in the package 'c' is called just once even if 'c' is
// imported twice.

import _ "a"
import _ "b"

func main() {
}
