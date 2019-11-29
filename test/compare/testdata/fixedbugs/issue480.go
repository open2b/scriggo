// errorcheck

package main

func f1(a int, a bool) {} // ERROR `duplicate argument a`

func f2() (b int, b bool) { return 0, false } // ERROR `duplicate argument b`

func f1(e int, e bool, c int, d string) {} // ERROR `duplicate argument e`

func main() {
	_ = func(c int, c bool) {}                     // ERROR `duplicate argument c`
	_ = func() (d int, d bool) { return 0, false } // ERROR `duplicate argument d`
}
