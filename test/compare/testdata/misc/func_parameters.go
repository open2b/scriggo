// errorcheck

package main

func main() {}

func f1()                    {}
func f2(int)                 {}
func f3(int)                 {}
func f4(a int)               {}
func f5(a int)               {}
func f6(int, string)         {}
func f7(int, ...string)      {}
func f8(a int, b string)     {}
func f9(a, b string)         {}
func f10(a int, b ...string) {}
func f11(a int, b ...string) {}

func f12() () {}
func f13() (int) { return 0 }
func f14() (a int) { return 0 }

func f(a int, b)           {} // ERROR `mixed named and unnamed function parameters`
func f(a int, b...)        {} // ERROR `mixed named and unnamed function parameters`
func f(...)                {} // ERROR `final argument in variadic function missing type`
func f(*int, x string)     {} // ERROR `mixed named and unnamed function parameters`
func f(...int, b string)   {} // ERROR `mixed named and unnamed function parameters`
func f(a ...int, b string) {} // ERROR `cannot use ... with non-final parameter a`
func f(...int, string)     {} // ERROR `cannot use ... with non-final parameter`
func f(a int, b string,,)  {} // ERROR `unexpected comma, expecting )`

func f() (a int, b) {} // ERROR `mixed named and unnamed function parameters`
func f() (...int)   {} // ERROR `cannot use ... in receiver or result parameter list`
func f() (a ...int) {} // ERROR `cannot use ... in receiver or result parameter list`
