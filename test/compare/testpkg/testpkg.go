// package testpkg is a package containing declarations used in tests.
package testpkg

import "fmt"

func G00()          {}
func G01() int      { return 40 }
func G10(a int)     {}
func G11(a int) int { return a + 33 }

func Sum(a, b int) int {
	return a + b
}
func StringLen(s string) int {
	return len(s)
}
func Pair() (int, int) {
	return 42, 33
}
func Inc(a int) int {
	return a + 1
}
func Dec(a int) int {
	return a - 1
}
func Swap(a int, b string) (string, int) {
	return b, a
}
func PrintString(s string) {
	fmt.Print(s)
}
func PrintInt(i int) {
	fmt.Print(i)
}

var A int = 20
var B int = 42

type TestPointInt struct {
	A, B int
}

func GetPoint() TestPointInt {
	return TestPointInt{A: 5, B: 42}
}

var Center = TestPointInt{A: 5, B: 42}

func NewT(a int) T {
	return T(a)
}

type T int

type Complex128 complex128

func SayHello() {
	fmt.Println("Hello, world!")
}
