// runcmp

package main

import "fmt"

func main() {
	s := []int{1, 2, 3}
	fmt.Println(s[0:0:0], cap(s[0:0:0]))
	fmt.Println(s[1:2:3], cap(s[1:2:3]))
	fmt.Println(s[1:2:2], cap(s[1:2:2]))
}
