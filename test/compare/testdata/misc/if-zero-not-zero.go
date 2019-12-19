// run

package main

import "fmt"

func main() {

	v := 0
	if v == 0 {
		fmt.Println("ok: v == 0")
	}
	if 0 == v {
		fmt.Println("ok: 0 == v")
	}
	if v != 0 {
		panic("v != 0")
	}
	if 0 != v {
		panic("0 != v")
	}

	v = 1
	if v == 0 {
		panic("v == 0")
	}
	if 0 == v {
		panic("0 == v")
	}
	if v != 0 {
		fmt.Println("ok: v != 0")
	}
	if 0 != v {
		fmt.Println("ok: 0 != v")
	}
}
