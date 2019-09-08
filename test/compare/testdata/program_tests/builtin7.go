// run

package main

import "fmt"

func main() {
	src := []int{10, 20, 30}
	dst := []int{1, 2, 3}
	n := copy(dst, src)
	fmt.Println("dst:", dst, "n:", n)
	dst2 := []int{1, 2}
	copy(dst2, src)
	fmt.Println("dst2:", dst2)
}
