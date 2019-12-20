// run

package main

import "fmt"

func main() {
	f := func() string {
		fmt.Println("f()")
		return ""
	}
	_, _ = f(), f()
}
