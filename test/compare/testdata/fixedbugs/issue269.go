// run

package main

import "fmt"

func main() {
	switch i := interface{}(true); {
	case i:
		fmt.Print("1")
	case false:
		fmt.Print("2")
	default:
		fmt.Print("3")
	}

	switch i := interface{}(true); {
	case i:
		fmt.Print("i")
	}

	switch interface{}(true) {
	case true:
	}

	switch true {
	case interface{}(true):
	}

	switch interface{}(true) {
	case interface{}(true):
	}

	switch true {
	case true:
	}

	switch float64(10) {
	case 10:
	}

}
