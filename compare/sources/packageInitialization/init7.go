package main

import "fmt"

var A = B + 20
var B = C + 49
var C = C1 + 100

const C1 = C2 + 43
const C2 = 300

func main() {
	fmt.Print(A, B, C, C1, C2)
}
