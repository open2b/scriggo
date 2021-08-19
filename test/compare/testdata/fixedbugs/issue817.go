// errorcheck

package main

type I interface {
	M() // ERROR `non-empty interfaces are not supported in this release of Scriggo`
}

func main() {}
