// run

package main

import "fmt"

func main() {
	var x int = 1
	{
		var x int = x + 1
		fmt.Print(x)
	}
}
