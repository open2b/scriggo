// run

package main

import "fmt"

func g() (string, int, string) {
	return "hello %d, %q", 3, "hi"
}

func main() {
	s := fmt.Sprintf(g())
	fmt.Println(s)
}
