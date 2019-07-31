

package main

var A string = "A"

func F() int {
	A := 10
	return A
}

var B = F()

func main() {
}
