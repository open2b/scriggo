// run

package main

import "fmt"

func g() string {
	return "hello %d, %q"
}

func main() {
	s := fmt.Sprintf(g())
	fmt.Println(s)
}
