// errorcheck

package main

var v2 [10]int = nil // ERROR `cannot use nil as type [10]int in assignment`

func main() {}
