// errorcheck

package main

func main() {
	_ = ([]int){} // ERROR `syntax error: cannot parenthesize type in composite literal`
}