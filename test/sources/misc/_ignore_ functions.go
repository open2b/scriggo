//+build ignore

package main

import "fmt"

func main() {
	{
		f := func() {
			fmt.Println("hi!")
		}
		f()
	}
}
