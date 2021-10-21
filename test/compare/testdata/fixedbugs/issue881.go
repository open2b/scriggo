// errorcheck

package main

import (
	"fmt"
	"fmt" // ERROR "fmt redeclared as imported package name"
)

func main() {
	_ = fmt.Print
}
