// run

package main

import "fmt"

func f(flag bool) {
	if flag {
		fmt.Println("flag is true")
	} else {
		fmt.Println("flag is false")
	}
}

func main() {
	f(true)
	f(false)
}
