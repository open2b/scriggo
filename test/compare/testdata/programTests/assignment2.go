// run

package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := exec.Cmd{}
	c.Dir = "/home/user/"
	c.Path = "git/prg"
	fmt.Println(c)
	fmt.Println(c.Dir)
	fmt.Println(c.Path)
}
