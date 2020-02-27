// run

package main

import "fmt"

func F() int {
	return 300
}

func G(d interface{}) (int, int) {
	return F(), 200
}

func main() {
	fmt.Println(G(100))
}
