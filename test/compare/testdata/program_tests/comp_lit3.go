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
	path := c.Path
	fmt.Printf("Path is %q", path)
}
