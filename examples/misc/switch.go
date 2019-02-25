// +build ignore

package main

import "fmt"

func main() {

	//-----

	a := 10
	switch a + 1 {
	case 11:
		fmt.Println(11)
	}

	//-----

	switch false {
	}

	//-----

	switch interface{}(4).(type) {
	case int:
		fmt.Println("is an int!")
	}

	//-----

	switch 4 {
	default:
		fmt.Println("unknown")
	case 2:
		fmt.Println("is 2")
	case 3:
		fmt.Println("is 3")
	case 4:
		fmt.Println("is 4")
	}

	//-----
}
