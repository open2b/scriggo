// errorcheck

package main

import (
	. "fmt"
)

func main() {
	var Printf int // ERROR `imported and not used: "fmt"`
	_ = Printf
}
