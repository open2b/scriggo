// run

package main

import "fmt"

func main() {
	f := func() {
		fmt.Println("end")
	}
	defer f()
	fmt.Println("start")
}
