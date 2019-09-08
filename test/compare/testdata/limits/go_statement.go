// errorcheck --disallowGoStatement

package main

func main() {
	go func() { println("go") }() // ERROR `"go" statement not available`
}
