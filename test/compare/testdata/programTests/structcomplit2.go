// run

package main

import (
	"errors"
	"fmt"
	"os/exec"
)

func main() {
	e := exec.Error{"errorName", errors.New("error")}
	fmt.Println(e)
}
