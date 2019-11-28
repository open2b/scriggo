// errorcheck

package main

type Int int

func main() {

	var i int
	var ii Int
	var b bool

	// constant + constant.
	_ = 1 + 2
	_ = 1 + int(2)
	_ = int(1) + 2
	_ = int(1) + int(2)
	_ = true + false             // ERROR `invalid operation: true + false (operator + not defined on bool)`
	_ = bool(true) + bool(false) // ERROR `invalid operation: bool(true) + bool(false) (operator + not defined on bool)`

	// untyped + untyped.
	_ = 1 + (1 << i)
	_ = 1 + ('a' << i)
	_ = 1 + (1.0 << i)      // ERROR `invalid operation: 1.0 << i (shift of type float64)`
	_ = 1 + (0i << i)       // ERROR `invalid operation: 0i << i (shift of type complex128)`
	_ = (1 == i) + (2 == i) // ERROR `invalid operation: (1 == i) + (2 == i) (operator + not defined on bool)`

	_ = (1 << i) + 1
	_ = ('a' << i) + 1
	_ = (1.0 << i) + 1 // ERROR `invalid operation: 1.0 << i (shift of type float64)`
	_ = (0i << i) + 1  // ERROR `invalid operation: 0i << i (shift of type complex128)`

	// typed + untyped.
	_ = i + 2
	_ = i + (1 << i)
	_ = i + ('a' << i)
	_ = i + (1.0 << i)
	_ = i + (0i << i)
	_ = int(1) + (1 << i)
	_ = int(1) + ('a' << i)
	_ = int(1) + (1.0 << i)
	_ = int(1) + (0i << i)
	_ = ii + (1 << i)
	_ = ii + ('a' << i)
	_ = ii + (1.0 << i)
	_ = ii + (0i << i)
	_ = bool(true) + (2 == i) // ERROR `invalid operation: bool(true) + (2 == i) (operator + not defined on bool)`

	// untyped + typed.
	_ = 2 + i
	_ = (1 << i) + i
	_ = ('a' << i) + i
	_ = (1.0 << i) + i
	_ = (0i << i) + i
	_ = (1 << i) + int(1)
	_ = ('a' << i) + int(1)
	_ = (1.0 << i) + int(1)
	_ = (0i << i) + int(1)
	_ = (1 << i) + ii
	_ = ('a' << i) + ii
	_ = (1.0 << i) + ii
	_ = (0i << i) + ii
	_ = (2 == i) + bool(false) // ERROR `invalid operation: (2 == i) + bool(false) (operator + not defined on bool)`

	// typed + typed.
	_ = i + i
	_ = i + int(2)
	_ = int(1) + i
	_ = int(1) + int(2)
	_ = ii == ii
	_ = bool(true) + bool(false) // ERROR `invalid operation: bool(true) + bool(false) (operator + not defined on bool)`

	// shifts.
	_ = 1 << 2
	_ = 1 << int(2)
	_ = 1 << (1 << i)
	_ = 1 << i
	_ = 1 << ii
	_ = int(1) << 2
	_ = int(1) << int(2)
	_ = int(1) << (1 << i)
	_ = int(1) << i
	_ = int(1) << ii
	_ = (1 << i) << 2
	_ = (1 << i) << int(2)
	_ = (1 << i) << (1 << i)
	_ = (1 << i) << i
	_ = (1 << i) << ii
	_ = i << 2
	_ = i << int(2)
	_ = i << (1 << i)
	_ = i << i
	_ = i << ii
	_ = ii << 2
	_ = ii << int(2)
	_ = ii << (1 << i)
	_ = ii << i
	_ = ii << ii

	// booleans.
	_ = true == false
	_ = true == bool(false)
	_ = true == (1 == 2)
	_ = true == b
	_ = bool(true) == false
	_ = bool(true) == bool(false)
	_ = bool(true) == (1 == 2)
	_ = bool(true) == b
	_ = (1 == 2) == false
	_ = (1 == 2) == bool(false)
	_ = (1 == 2) == (1 == 2)
	_ = (1 == 2) == b
	_ = b == false
	_ = b == bool(false)
	_ = b == (1 == 2)
	_ = b == b

	// less on booleans.
	_ = true < false             // ERROR `invalid operation: true < false (operator < not defined on bool)`
	_ = true < bool(false)       // ERROR `invalid operation: true < bool(false) (operator < not defined on bool)`
	_ = true < (1 == 2)          // ERROR `invalid operation: true < (1 == 2) (operator < not defined on bool)`
	_ = true < b                 // ERROR `invalid operation: true < b (operator < not defined on bool)`
	_ = bool(true) < false       // ERROR `invalid operation: bool(true) < false (operator < not defined on bool)`
	_ = bool(true) < bool(false) // ERROR `invalid operation: bool(true) < bool(false) (operator < not defined on bool)`
	_ = bool(true) < (1 == 2)    // ERROR `invalid operation: bool(true) < (1 == 2) (operator < not defined on bool)`
	_ = bool(true) < b           // ERROR `invalid operation: bool(true) < b (operator < not defined on bool)`
	_ = (1 == 2) < false         // ERROR `invalid operation: (1 == 2) < false (operator < not defined on bool)`
	_ = (1 == 2) < bool(false)   // ERROR `invalid operation: (1 == 2) < bool(false) (operator < not defined on bool)`
	_ = (1 == 2) < (1 == 2)      // ERROR `invalid operation: (1 == 2) < (1 == 2) (operator < not defined on bool)`
	_ = (1 == 2) < b             // ERROR `invalid operation: (1 == 2) < b (operator < not defined on bool)`
	_ = b < false                // ERROR `invalid operation: b < false (operator < not defined on bool)`
	_ = b < bool(false)          // ERROR `invalid operation: b < bool(false) (operator < not defined on bool)`
	_ = b < (1 == 2)             // ERROR `invalid operation: b < (1 == 2) (operator < not defined on bool)`
	_ = b < b                    // ERROR `invalid operation: b < b (operator < not defined on bool)`

	// nil.
	_ = nil == nil      // ERROR `invalid operation: nil == nil (operator == not defined on nil)`
	_ = nil == 2        // ERROR `cannot convert nil to type int`
	_ = nil == int(2)   // ERROR `cannot convert nil to type int`
	_ = nil == (2 << i) // ERROR `cannot convert nil to type int`
	_ = nil == i        // ERROR `cannot convert nil to type int`
	_ = nil == ii       // ERROR "cannot convert nil to type Int"
	_ = 1 == nil        // ERROR `cannot convert nil to type int`
	_ = int(1) == nil   // ERROR `cannot convert nil to type int`
	_ = (2 << i) == nil // ERROR `cannot convert nil to type int`
	_ = i == nil        // ERROR `cannot convert nil to type int`
	_ = ii == nil       // ERROR "cannot convert nil to type Int"
	_ = nil < nil       // ERROR `invalid operation: nil < nil (operator < not defined on nil)`
	_ = nil < 2         // ERROR `cannot convert nil to type int`
	_ = nil < int(2)    // ERROR `cannot convert nil to type int`
	_ = nil < (2 << i)  // ERROR `cannot convert nil to type int`
	_ = nil < i         // ERROR `cannot convert nil to type int`
	_ = nil < ii        // ERROR "cannot convert nil to type Int"
	_ = 1 < nil         // ERROR `cannot convert nil to type int`
	_ = int(1) < nil    // ERROR `cannot convert nil to type int`
	_ = (1 << i) < nil  // ERROR `cannot convert nil to type int`
	_ = i < nil         // ERROR `cannot convert nil to type int`
	_ = ii < nil        // ERROR "cannot convert nil to type Int"

	// interfaces.
	_ = interface{}(1) == interface{}(1)
	_ = 1 == interface{}(1)
	_ = int(1) == interface{}(1)
	_ = (1 << i) == interface{}(1)
	_ = i == interface{}(1)
	_ = interface{}(1) == 1
	_ = interface{}(1) == int(1)
	_ = interface{}(1) == (1 << i)
	_ = interface{}(1) == i

	// Operation +
	_ = 1.2 + 2.3
	_ = 1.1 + float64(2.3)
	_ = float64(1.2) + 2.3
	_ = float64(1.2) + float64(2.3)
	_ = 1i + 2i
	_ = 1i + complex128(2i)
	_ = complex128(1i) + 2i
	_ = complex128(1i) + complex128(2i)

	// Operation *
	_ = 1 * 2
	_ = 1 * int(2)
	_ = int(1) * 2
	_ = int(1) * int(2)
	_ = 1.2 * 2.3
	_ = 1.1 * float64(2.3)
	_ = float64(1.2) * 2.3
	_ = float64(1.2) * float64(2.3)
	_ = 1i * 2i
	_ = 1i * complex128(2i)
	_ = complex128(1i) * 2i
	_ = complex128(1i) * complex128(2i)

	// Operation /
	_ = 1 / 2
	_ = 1 / int(2)
	_ = int(1) / 2
	_ = int(1) / int(2)
	_ = 1.2 / 2.3
	_ = 1.1 / float64(2.3)
	_ = float64(1.2) / 2.3
	_ = float64(1.2) / float64(2.3)
	_ = 1i / 2i
	_ = 1i / complex128(2i)
	_ = complex128(1i) / 2i
	_ = complex128(1i) / complex128(2i)

	// Operation %
	_ = 1 % 2
	_ = 1 % int(2)
	_ = int(1) % 2
	_ = int(1) % int(2)
	_ = 1.2 % 2.3                       // ERROR `invalid operation: 1.2 % 2.3 (floating-point % operation)`
	_ = 1.1 % float64(2.3)              // ERROR `invalid operation: 1.1 % float64(2.3) (operator % not defined on float64)`
	_ = float64(1.2) % 2.3              // ERROR `invalid operation: float64(1.2) % 2.3 (operator % not defined on float64)`
	_ = float64(1.2) % float64(2.3)     // ERROR `invalid operation: float64(1.2) % float64(2.3) (operator % not defined on float64)`
	_ = 1i % 2i                         // ERROR `invalid operation: 1i % 2i (operator % not defined on untyped complex)`
	_ = 1i % complex128(2i)             // ERROR `invalid operation: 1i % complex128(2i) (operator % not defined on complex128)`
	_ = complex128(1i) % 2i             // ERROR `invalid operation: complex128(1i) % 2i (operator % not defined on complex128)`
	_ = complex128(1i) % complex128(2i) // ERROR `invalid operation: complex128(1i) % complex128(2i) (operator % not defined on complex128)`

}
