// errorcheck

package main

var A int = 0

func main() {
	A = "" // ERROR `cannot use "" (type string) as type int in assignment`
}
