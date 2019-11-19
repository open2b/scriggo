// errorcheck

package main

func main() {
	x := nil ; _ = x // ERROR `use of untyped nil`
}
