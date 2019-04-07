package vm

import (
	"bytes"
	"reflect"
	"scrigo"
	"strconv"
	"testing"

	"github.com/d5/tengo/script"
)

// Test:
//
// x := 0
// for i := 0; i < 5; i++ {
//     x = inc(x)
// }

var x = 0
var tot = 127

func BenchmarkGoCall(b *testing.B) {
	for n := 0; n < b.N; n++ {
		x = 0
		for i := 0; i < tot; i++ {
			x = inc(x)
		}
	}
}

//go:noinline
func inc(x int) int {
	return x + 1
}

var scrigovmT1 *VM

func init() {

	pkg := NewPackage("main")

	inc := pkg.NewFunction("inc", []Type{TypeInt}, []Type{TypeInt}, false)
	fb := inc.Builder()
	fb.Add(true, 1, 1, 0, reflect.Int)
	fb.Move(false, 1, 0, reflect.Int)
	fb.Return()
	fb.End()

	main := pkg.NewFunction("main", nil, nil, false)

	fb = main.Builder()
	cInc := fb.MakeInterfaceConstant(inc)
	fb.Move(true, 0, 0, reflect.Int) // i := 0
	fb.Move(true, 0, 1, reflect.Int) // x := 0
	fb.SetLabel()
	fb.If(true, 0, ConditionLess, int8(tot), reflect.Int)
	fb.Goto(3)
	fb.SetLabel()
	// x = inc(x)
	fb.Move(false, 1, 3, reflect.Int)
	fb.Call(cInc, StackShift{2})
	fb.Move(false, 2, 1, reflect.Int)
	//
	fb.Add(true, 0, 1, 0, reflect.Int) // i++
	fb.Goto(1)
	fb.SetLabel()
	fb.Return()
	fb.End()

	scrigovmT1 = New(pkg)

	DebugTraceExecution = false
}

func BenchmarkScrigoVMCall(b *testing.B) {
	var err error
	for n := 0; n < b.N; n++ {
		_, err = scrigovmT1.Run("main")
		if err != nil {
			panic(err)
		}
		scrigovmT1.Reset()
	}
}

var tengoT1 *script.Compiled

func init() {
	var code = `
	inc := func(x) {
    	return x + 1
	}
	for i := 0; i < ` + strconv.Itoa(tot) + `; i++ {
		i = inc(i)
	}
`
	s := script.New([]byte(code))
	var err error
	tengoT1, err = s.Compile()
	if err != nil {
		panic(err)
	}
}

func BenchmarkTengoCall(b *testing.B) {
	var err error
	for n := 0; n < b.N; n++ {
		err = tengoT1.Run()
		if err != nil {
			panic(err)
		}
	}
}

var scrigoT1 *scrigo.Script

func init() {
	src := `
	inc := func(x int) int { return x + 1 }
	var i = 0
	for i := 0; i < ` + strconv.Itoa(tot) + `; i++ {
		i = inc(i)
	}
	`
	r := bytes.NewReader([]byte(src))
	var err error
	scrigoT1, err = scrigo.CompileScript(r, nil)
	if err != nil {
		panic(err)
	}
}

func BenchmarkScrigoCall(b *testing.B) {
	var err error
	for n := 0; n < b.N; n++ {
		_, err = scrigo.ExecuteScript(scrigoT1, nil)
		if err != nil {
			panic(err)
		}
	}
}
