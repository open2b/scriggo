// run

package main

import (
	"fmt"
	"os/exec"
)

func s() string {
	fmt.Print("s()")
	return ""
}

func main() {
	_ = exec.Error{Name: s()}
}
