// run

package main

import "fmt"

func main() {

	fmt.Printf("start-")
	for _ = range []int{} {
		fmt.Print("looping!")
	}
	fmt.Printf("-end")

}
