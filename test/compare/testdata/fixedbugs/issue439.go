// run

package main

import (
	"strconv"
)

func g() *error {
	var x error
	return &x
}

func main() {
	_, *g() = strconv.Atoi("3")
}
