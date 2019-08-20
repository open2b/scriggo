// run

package main

import (
	"fmt"
	"os/exec"
)

func main() {
	e := &exec.Error{Name: "errorName"}
	fmt.Println(e.Name)
}
