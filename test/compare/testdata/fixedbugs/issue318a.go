// run

package main

import (
	"fmt"
	"strings"
)

func getBuilder() strings.Builder {
	return strings.Builder{}
}

func main() {
	b := getBuilder()
	fmt.Println(b)
}
