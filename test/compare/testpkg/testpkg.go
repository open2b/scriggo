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

type I interface {
	M()
}

func SayHello() {
	fmt.Println("Hello, world!")
}

func F1() { fmt.Println("f1") }

func F2(i int) { fmt.Printf("f2 %d\n", i) }

func F3(f float64) { fmt.Printf("f3 %f\n", f) }

func F4(s string) { fmt.Printf("f4 %q\n", s) }

func F5(s []int) { fmt.Printf("f5 %v\n", s) }

func F6(a, b int) { fmt.Printf("f6 %d %d\n", a, b) }

func F7(a int, b float64, c string) { fmt.Printf("f7 %d %f %q\n", a, b, c) }

func F8(a ...int) { fmt.Printf("f8 %v\n", a) }

func F9(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }

func F10() int { fmt.Println("f10"); return 5 }

func F11(a string, b float64, c []string) (string, error) {
	fmt.Printf("f11 %q %f %v\n", a, b, c)
	return "", nil
}
