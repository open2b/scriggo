// run

package main

import "fmt"

func main() {
	s := 10
	v := 3 >> s
	_ = v
	n := 1
	fmt.Println(300 >> n)
	fmt.Println(300 << n)
	fmt.Println(uint8(20) << n)
}
