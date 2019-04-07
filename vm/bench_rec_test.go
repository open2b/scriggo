package vm

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/d5/tengo/script"
)

// Test:
//
// func fib(n, a, b int) int {
//     if n == 0 {
//         return a
//     } else if n == 1 {
//         return b
//     }
//     return fib(n-1, b, a+b)
// }

var fibN = 35

func BenchmarkGoCallRec(b *testing.B) {
	for n := 0; n < b.N; n++ {
		x = fib(fibN, 0, 1)
	}
}

func fib(n, a, b int) int {
	if n == 0 {
		return a
	} else if n == 1 {
		return b
	}
	return fib(n-1, b, a+b)
}

var scrigovmT2 *VM

func init() {
	pkg := NewPackage("main")

	fib := pkg.NewFunction("fib", []Type{TypeInt, TypeInt, TypeInt}, []Type{TypeInt}, false)
	fb := fib.Builder()

	fb.If(true, 1, ConditionEqual, 0, reflect.Int) // if n == 0
	fb.Goto(1)
	fb.Move(false, 2, 0, reflect.Int) // return a
	fb.Return()
	fb.SetLabel()
	fb.If(true, 1, ConditionEqual, 1, reflect.Int) // if n == 1
	fb.Goto(2)
	fb.Move(false, 3, 0, reflect.Int) // return b
	fb.Return()
	fb.SetLabel()
	fb.Sub(true, 1, 1, 1, reflect.Int)  // n = n-1
	fb.Move(false, 3, 4, reflect.Int)   // t = b
	fb.Add(false, 2, 3, 3, reflect.Int) // b = a+b
	fb.Move(false, 4, 2, reflect.Int)   // a = t
	fb.TailCall(NoRegister)
	fb.End()

	main := pkg.NewFunction("main", nil, nil, false)
	fb = main.Builder()
	cFib := fb.MakeInterfaceConstant(fib)
	fb.Move(true, int8(fibN), 1, reflect.Int) // n := fibN
	fb.Move(true, 0, 2, reflect.Int)          // a := 0
	fb.Move(true, 1, 3, reflect.Int)          // b := 1
	fb.Call(cFib, StackShift{0})
	fb.Return()
	fb.End()

	scrigovmT2 = New(pkg)

	DebugTraceExecution = false
}

func BenchmarkScrigoVMCallRec(b *testing.B) {
	var err error
	for n := 0; n < b.N; n++ {
		_, err = scrigovmT2.Run("main")
		if err != nil {
			panic(err)
		}
		scrigovmT2.Reset()
	}
}

var tengoT2 *script.Compiled

func init() {
	var code = `
	fib := func(n, a, b) {
		if n == 0 {
			return a
		} else if n == 1 {
			return b
		}
		return fib(n-1, b, a+b)
	}
	fib(` + strconv.Itoa(fibN) + `, 0, 1)
`

	s := script.New([]byte(code))
	var err error
	tengoT2, err = s.Compile()
	if err != nil {
		panic(err)
	}
}

func BenchmarkTengoCallRec(b *testing.B) {
	var err error
	for n := 0; n < b.N; n++ {
		err = tengoT2.Run()
		if err != nil {
			panic(err)
		}
	}
}
