// run

package main

import "fmt"

func main() {

	// Testing predefined function literals.
	p := fmt.Println
	p("a")
	p("a", "b")
	p()
	p([]int{1, 2, 3})
	p([]interface{}{1, 2, 3}...)
	p([]interface{}(nil)...)

	// Testing function literals defined in Scriggo.
	f1 := func() {}
	f1()

	f2 := func(a int) { fmt.Println(a) }
	f2(10)

	f3 := func(args ...int) { fmt.Println(len(args), args) }
	f3(1, 2, 3)
	f3([]int{1, 2, 3}...)
	f3()
	f3([]int{}...)
	f3([]int(nil)...)

	sum := func(args ...int) int {
		sum := 0
		for _, v := range args {
			sum += v
		}
		return sum
	}

	fmt.Println(sum(1, 2, 3))
	fmt.Println(sum())
	fmt.Println(sum([]int{1, 2, 3, 4, 5}...))

}
