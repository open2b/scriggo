// run

package main

import "fmt"

func test1() {
	f1 := func() { fmt.Print("f1") }
	_ = &f1
	f1()
}

func test2() {
	f1 := func() {}
	_ = &f1
	f1()
}

func test3() {
	var indirectVar [3]int
	func() {
		a := indirectVar
		_ = a
	}()
	indirectVar = [3]int{1, 2, 3}
}

func test4() {
	var a [2]int
	fmt.Printf("%v %T\n", &a, &a)
	a[1] = 2
}

func main() {
	test1()
	test2()
	test3()
	test4()
}
