// run

package main

import "fmt"

func f() string {
	fmt.Println("f()")
	return ""
}

func main() {
	_, _ = f(), f()
}
