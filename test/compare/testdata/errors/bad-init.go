// errorcheck

package main

func init() {
}

func init(a int) { // ERROR `func init must have no arguments and no return values`
}

func main() {
}
