// errorcheck

package main

import (
	"testpkg"
)

type uint uint8

var a, int32 = h()

func h() (int16, int16) { return 0, 0 }

var c, int = 0, 0

const d, string = "", ""

func bool() {}

func f1(a int)       {} // ERROR `int is not a type`
func f2(a string)    {} // ERROR `string is not a type`
func f3(a bool)      {} // ERROR `bool is not a type`
func f4(a int32)     {} // ERROR `int32 is not a type`
func f5(a uint)      {}
func f6(a testpkg.T) {}

func float32(a float32) {} // ERROR `float32 is not a type`

func f7() int        { return 0 }     // ERROR `int is not a type`
func f8() string     { return "" }    // ERROR `string is not a type`
func f9() bool       { return false } // ERROR `bool is not a type`
func f10() int32     { return 0 }     // ERROR `int32 is not a type`
func f11() uint      { return 0 }
func f12() testpkg.T { return testpkg.T(0) }

func float64() float64 { return 0 } // ERROR `float64 is not a type`

func main() {}
