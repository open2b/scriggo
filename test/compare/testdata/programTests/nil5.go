// run

package main

import (
	"fmt"
)

func f(i interface{}) {
	fmt.Println(i)
}

func main() {
	f(10)
	f(nil)
}
