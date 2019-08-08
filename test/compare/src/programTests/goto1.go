// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("out1,")
	{
		fmt.Print("goto,")
		goto out
		fmt.Print("error")
	}
out:
	fmt.Print("out2")
}
