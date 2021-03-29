package compiler

import (
	"reflect"
	"testing"

	"github.com/open2b/scriggo/runtime"
)

func TestFinalize(t *testing.T) {

	intType := reflect.TypeOf(0)
	f := newFunction("main", "f", reflect.FuncOf([]reflect.Type{}, []reflect.Type{intType}, false), "", nil)
	fb := newBuilder(f, "")
	fb.allocRegister(intRegister, 1)
	fb.allocRegister(generalRegister, 1)
	fb.allocRegister(generalRegister, 2)
	fb.emitNew(intType, 1)
	fb.emitTypify(true, intType, 0, -1)
	dType := reflect.FuncOf(nil, nil, false)
	d := fb.emitFunc(2, dType, nil, false, 0)
	fb.emitDefer(2, runtime.NoVariadicArgs, runtime.StackShift{1, 0, 0, 2}, runtime.StackShift{}, dType)
	fb.emitMove(false, -1, 1, reflect.Int, true)
	f.FinalRegs = [][2]int8{{1, 1}}
	fb.end()

	d.VarRefs = []int16{-1}
	fb = newBuilder(d, "")
	fb.emitSetVar(true, 5, 0, reflect.Int)
	fb.end()

	main := newFunction("main", "main", reflect.FuncOf(nil, nil, false), "", nil)
	fb = newBuilder(main, "")
	fb.allocRegister(intRegister, 1)
	fb.allocRegister(generalRegister, 1)
	fRef := fb.addFunction(f)
	fb.emitCallFunc(int8(fRef), runtime.StackShift{}, nil)
	fb.emitTypify(false, intType, 1, 1)
	fb.emitPrint(1)
	fb.end()

	asm := DisassembleFunction(f, []Global{}, 0)
	println(string(asm))

	vm := runtime.NewVM()
	c, err := vm.Run(main, nil, []interface{}{})
	if err != nil {
		t.Fatalf("\nerror: %v", err)
	}
	println("\nexit code:", c)
}
