// errorcheck

package main

import "fmt"

func typeof(i interface{}) string { fmt.Sprintf("%T", i) } // ERROR `missing return at end of function`

func e() int { } // ERROR `missing return at end of function`

func f() int { println("hello") } // ERROR `missing return at end of function`

func g() (int, int) { println("hello") ; println("hello") } // ERROR `missing return at end of function`

func h() string { if true { return "true!" } } // ERROR `missing return at end of function`

func main() {
	typeof := func(i interface{}) string { fmt.Sprintf("%T", i) } // ERROR `missing return at end of function`

	e := func() int { } // ERROR `missing return at end of function`

	f := func() int { println("hello") } // ERROR `missing return at end of function`

	g := func() (int, int) { println("hello") ; println("hello") } // ERROR `missing return at end of function`

	h := func() string { if true { return "true!" } } // ERROR `missing return at end of function`
}