// Package testpkg is a package containing declarations used in tests.
package testpkg

import (
	"fmt"

	"github.com/open2b/scriggo/native"
)

const (
	// C1 is a constant used in tests.
	C1 = "a\t|\"c"
	// C2 is a constant used in tests.
	C2 = true
	// C3 is a constant used in tests.
	C3 = 1982717381
	// C4 is a constant used in tests.
	C4 = 1.319382
	// C5 is a constant used in tests.
	C5 = 3.90i
	// C6 is a constant used in tests.
	C6 = 'a'
	// C7 is a constant used in tests.
	C7 = 1.0261 + 2.845i
	// C8 is a constant used in tests.
	C8 = 1 + 1.3 - 'b' + 1i
	// C9 is a constant used in tests.
	C9 = 1 + 0i
)

// G00 is a function used in tests.
func G00() {}

// G01 is a function used in tests.
func G01() int { return 40 }

// G10 is a function used in tests.
func G10(a int) {}

// G11 is a function used in tests.
func G11(a int) int { return a + 33 }

// Sum returns a + b.
func Sum(a, b int) int {
	return a + b
}

// StringLen returns len(s).
func StringLen(s string) int {
	return len(s)
}

// Pair returns a number pair used in tests.
func Pair() (int, int) {
	return 42, 33
}

// Inc returns a + 1.
func Inc(a int) int {
	return a + 1
}

// Dec returns a - 1.
func Dec(a int) int {
	return a - 1
}

// Swap returns its argument swapped.
func Swap(a int, b string) (string, int) {
	return b, a
}

// PrintString calls fmt.Print(s).
func PrintString(s string) {
	fmt.Print(s)
}

// PrintInt calls fmt.Print(i).
func PrintInt(i int) {
	fmt.Print(i)
}

var (
	// A is a constant used in tests.
	A int = 20
	// B is a constant used in tests.
	B int = 42
)

// TestPointInt is a type used in tests.
type TestPointInt struct {
	A, B int
}

// GetPoint is a method used in tests.
func GetPoint() TestPointInt {
	return TestPointInt{A: 5, B: 42}
}

// Center is a value of TestPointInt used in tests.
var Center = TestPointInt{A: 5, B: 42}

// NewT returns a new T.
func NewT(a int) T {
	return T(a)
}

type (
	// T is a type used in tests.
	T int
	// S is a type used in tests.
	S string
	// Bool is a type used in tests.
	Bool bool
	// Int is a type used in tests.
	Int int
	// Float64 is a type used in tests.
	Float64 float64
	// Complex128 is a type used in tests.
	Complex128 complex128
	// String is a type used in tests.
	String string
)

// I is a type used in tests.
type I interface {
	M()
}

// SayHello is a type used in tests.
func SayHello() {
	fmt.Println("Hello, world!")
}

// F1 is a function used in tests.
func F1() { fmt.Println("f1") }

// F2 is a function used in tests.
func F2(i int) { fmt.Printf("f2 %d\n", i) }

// F3 is a function used in tests.
func F3(f float64) { fmt.Printf("f3 %f\n", f) }

// F4 is a function used in tests.
func F4(s string) { fmt.Printf("f4 %q\n", s) }

// F5 is a function used in tests.
func F5(s []int) { fmt.Printf("f5 %v\n", s) }

// F6 is a function used in tests.
func F6(a, b int) { fmt.Printf("f6 %d %d\n", a, b) }

// F7 is a function used in tests.
func F7(a int, b float64, c string) { fmt.Printf("f7 %d %f %q\n", a, b, c) }

// F8 is a function used in tests.
func F8(a ...int) { fmt.Printf("f8 %v\n", a) }

// F9 is a function used in tests.
func F9(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }

// F10 is a function used in tests.
func F10() int { fmt.Println("f10"); return 5 }

// F11 is a function used in tests.
func F11(a string, b float64, c []string) (string, error) {
	fmt.Printf("f11 %q %f %v\n", a, b, c)
	return "", nil
}

// Fatal calls env.Fatal(v).
func Fatal(env native.Env, v interface{}) {
	env.Fatal(v)
}

// CallFunction calls f.
func CallFunction(f func()) {
	f()
}

// CallVariadicFunction is a function used in tests.
func CallVariadicFunction(f func(s string, n ...int)) {
	f("abc")
	f("abc", 5)
	f("abc", 5, 9, 12)
}

// ReturnFunction is a function used in tests.
func ReturnFunction() func(int) int {
	return func(i int) int { return i + 1 }
}

// RuntimeError is a function that causes a runtime error.
func RuntimeError() {
	var a = 0
	_ = 1 / a
}

// Method is a method used in tests.
func (t *T) Method() {}

// Value is a variable used in tests.
var Value T

// BooleanValue is a variable used in tests.
var BooleanValue bool

// S1 is a type used in tests.
type S1 struct{ F int }

// S2 is a type used in tests.
type S2 struct{ *S1 }

// St is a type used in tests.
type St struct{ F int }

// Sp is a type used in tests.
type Sp *St

// Sv is a variable used in tests.
var Sv St

// These types are used to test method values.

// Tv is a type used in tests.
type Tv int

// Mv is a method used in tests.
func (t Tv) Mv(a int) int { return int(t) + a }

// Tp is a type used in tests.
type Tp int

// Mp is a method used in tests.
func (tp *Tp) Mp(a int) int {
	if tp == nil {
		return 0
	}
	return int(*tp) + a
}

// These types are used to test truthful values.

// True is a type used in tests.
type True struct {
	T bool
}

// IsTrue is a method used in tests.
func (t True) IsTrue() bool {
	return t.T
}

// TruePtr is a type used in tests.
type TruePtr struct {
	T bool
}

// IsTrue is a method used in tests.
func (t *TruePtr) IsTrue() bool {
	return t.T
}
