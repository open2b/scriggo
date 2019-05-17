//+build ignore

package main

import "fmt"

func main() {
	{
		f := func() {}
		f()
	}

	{
		f := func() { fmt.Println("func-literal") }
		f()
	}
}
