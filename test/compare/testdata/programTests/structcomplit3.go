// run

package main

import (
	"fmt"
	"os/exec"
)

func main() {
	c := exec.Cmd{
		Dir:  "/home/user/",
		Path: "git/prg",
	}
	fmt.Println(c)
	fmt.Println(c.Dir)
	fmt.Println(c.Path)
}
