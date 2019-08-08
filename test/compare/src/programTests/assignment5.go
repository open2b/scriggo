// run

package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := exec.Cmd{
		Path: "oldPath",
	}
	fmt.Print(c.Path, ",")
	c.Path = "newPath"
	fmt.Print(c.Path)
}
