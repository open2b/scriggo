// run

package main

import "fmt"

func main() {
	defer func(s int) {
		fmt.Print(s)
	}(1)
	fmt.Print(0)
}
