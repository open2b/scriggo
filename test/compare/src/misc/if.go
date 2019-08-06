// run

package main

import "fmt"

func main() {
	if !false {
		if !!false {
		} else if !false {
			fmt.Println("here")
		}
	}
	a := 3
	if true {
		a = 4
	} else if true {
		fmt.Println("shouldn't print this...")
	}
	if a == 4 {
		fmt.Println("a is 4")
	}
}
