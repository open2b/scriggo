// run

package main

import "fmt"

type B [2]int

const c = len(B{})

func main() {
	fmt.Println(c)
}
