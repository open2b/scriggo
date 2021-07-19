// run

package main

import "fmt"

func f(s ...interface{}) {
	fmt.Println(s[0])
}

func g() (int, int) {
	return 1, 2
}

func main() {
	f(g())
}
