// run

package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := exec.Cmd{
		Path: "aPath",
	}
	fmt.Printf("Path is %q", c.Path)
}
