// run

package main

import "fmt"

func pair() (int, int) {
	return 1, 2
}

func f1(args ...int) {
	l := len(args)
	fmt.Println("f1:", l)
}

func f2(args ...int) {
	fmt.Println("f2:", args)
}

func main() {
	f1(pair())
	f2(pair())
	fmt.Println(pair())
	f1(4, 5)
	f1()
	f1(4, 5, 6)
	f2(12, 321, 321)
}
