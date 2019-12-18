// run

package main

import "fmt"

func main() {
	x := "external "
	{
		x := x + "internal"
		fmt.Print(x)
	}
}
