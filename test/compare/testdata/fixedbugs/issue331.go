// skip : https://github.com/open2b/scriggo/issues/331

// errorcheck

package main

func main() {
	_ = ([]int){} // ERROR `syntax error: cannot parenthesize type in composite literal`
}