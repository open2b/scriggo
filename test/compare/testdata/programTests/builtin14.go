// run

package main

import "fmt"

func main() {
	s := make([]int, 5, 10)
	fmt.Println("s:", s)
	fmt.Println("len:", len(s))
	fmt.Println("cap:", cap(s))
}
