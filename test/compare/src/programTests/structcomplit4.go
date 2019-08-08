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
		Args: []string{"arg1", "arg2", "arg3"},
	}
	fmt.Println(c)
	fmt.Println(c.Dir)
	fmt.Println(c.Path)
	fmt.Println(c.Args)
}
