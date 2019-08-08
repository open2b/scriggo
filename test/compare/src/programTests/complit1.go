// run

package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := exec.Cmd{}
	fmt.Printf("%+v", c)
}
