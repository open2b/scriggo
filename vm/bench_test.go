package vm

import (
	"bytes"
	"reflect"
	"scrigo"
	"strconv"
	"testing"

	"github.com/d5/tengo/script"
)

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

func BenchmarkScrigoVMCall(b *testing.B) {

	pkg := NewPackage("main")

	inc := pkg.NewFunction("inc", []Type{TypeInt}, []Type{TypeInt}, false)
	fb := inc.Builder()
	fb.Add(true, 0, 1, 0, reflect.Int)
	fb.Move(false, 0, 1, reflect.Int)
	fb.Return()
	fb.End()

	main := pkg.NewFunction("main", nil, nil, false)

	// x := 0
	// for i := 0; i < 5; i++ {
	//     x = inc(x)
	// }

	fb = main.Builder()
	cInc := fb.MakeInterfaceConstant(inc)
	fb.Move(true, 0, 0, reflect.Int) // i := 0
	fb.Move(true, 0, 1, reflect.Int) // x := 0
	fb.SetLabel()
	fb.If(true, 0, ConditionLess, int8(tot), reflect.Int)
	fb.Goto(3)
	fb.SetLabel()
	// x = inc(x)
	fb.Move(false, 1, 2, reflect.Int)
	fb.Call(cInc)
	fb.Move(false, 3, 1, reflect.Int)
	//
	fb.Add(true, 0, 1, 0, reflect.Int) // i++
	fb.Goto(1)
	fb.SetLabel()
	fb.Return()
	fb.End()

	vm1 := New(pkg)

	DebugTraceExecution = false

	for n := 0; n < b.N; n++ {
		_, err := vm1.Run("main")
		if err != nil {
			panic(err)
		}
		vm1.Reset()
	}
}

func BenchmarkTengoCall(b *testing.B) {

	var code = `
	inc := func(x) {
    	return x + 1
	}
	for i := 0; i < ` + strconv.Itoa(tot) + `; i++ {
		i = inc(i)
	}
`

	s := script.New([]byte(code))

	for n := 0; n < b.N; n++ {
		_, err := s.Run()
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkScrigoCall(b *testing.B) {

	src := `
	inc := func(x int) int { return x + 1 }
	var i = 0
	for i := 0; i < ` + strconv.Itoa(tot) + `; i++ {
		i = inc(i)
	}
	`

	r := bytes.NewReader([]byte(src))
	script, err := scrigo.CompileScript(r, nil)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = scrigo.ExecuteScript(script, nil)
		if err != nil {
			panic(err)
		}
	}

}
