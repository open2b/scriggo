// run

package main

import "fmt"

func f(s ...interface{}) {
	fmt.Println(s[0])
}

func g() (int, []int) {
	return 3, []int{1, 5}
}

func main() {
	f(g())
}
