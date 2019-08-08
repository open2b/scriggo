// compile

package main

import (
	"fmt"
)

func pair() (int, int) {
	return 1, 2
}

func variadic(args ...int) {
	fmt.Printf("got %d arguments: %v\n", len(args), args)
}

func main() {
	variadic()
	variadic(pair())
}
