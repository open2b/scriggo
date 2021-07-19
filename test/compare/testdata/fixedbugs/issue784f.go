// run

package main

import "fmt"

func f(s ...interface{}) {
	fmt.Printf("%T %T", s[0], s[1])
	for i, v := range s {
		fmt.Printf("%d: %v\n", i, v)
	}
}

func g() (int, []int) {
	fmt.Println("g called :)")
	return 3, []int{1, 5}
}

func main() {
	f(g())
	f(32, 12, 2)
	x := 1
	f("ciao", 2, x)
	f([]int{1, 2, 3}, []string{"xyz"})
}
