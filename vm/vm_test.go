package vm

import (
	"reflect"
	"scrigo/parser"
	"testing"
)

type reg struct {
	typ   Type
	r     int8
	value interface{}
}

var cases = map[string][]reg{

	`a := 10; _ = a`: []reg{
		{TypeInt, 0, int64(10)}, // a
	},
	`a := 5; b := 10; c := a + b; _ = c`: []reg{
		{TypeInt, 0, int64(5)},  // a
		{TypeInt, 1, int64(10)}, // b
		{TypeInt, 2, int64(15)}, // c
	},
	`a := 10; c := 0; if a > 5 { c = 1 } else { c = 2 }; _ = c`: []reg{
		{TypeInt, 0, int64(10)}, // a
		{TypeInt, 1, int64(1)},  // c
	},
	`a := 10; c := 0; if a <= 5 { c = 1 } else { c = 2 }; _ = c`: []reg{
		{TypeInt, 0, int64(10)}, // a
		{TypeInt, 1, int64(2)},  // c
	},
	`a := 10 + 20 - 4; _ = a`: []reg{
		{TypeInt, 0, int64(26)}, // a
	},
	`c := 0; if x := 1; x == 1 { c = 1 } else { c = 2 }; _ = c`: []reg{
		{TypeInt, 0, int64(1)}, // c
	},
	// `a := "s"; _ = a`: []reg{
	// 	{TypeString, 0, "s"},
	// },
	`a := []int{1,2,4}; _ = a`: []reg{
		{TypeIface, 0, []int{1, 2, 4}},
	},
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for src, registers := range cases {
		fullSrc := "package main\nfunc main(){\n" + src + "\n}\n"
		r := parser.MapReader{"/test.go": []byte(fullSrc)}
		comp := NewCompiler(r, nil)
		pkg, err := comp.Compile("/test.go")
		if err != nil {
			t.Errorf("source: %q, compiler error: %s", src, err)
			continue
		}
		vm := New(pkg)
		_, err = vm.Run("main")
		if err != nil {
			t.Errorf("source: %q, execution error: %s", src, err)
			continue
		}
		for _, reg := range registers {
			var got interface{}
			switch reg.typ {
			case TypeFloat:
				got = vm.float(reg.r)
			case TypeIface:
				got = vm.value(reg.r)
			case TypeInt:
				got = vm.int(reg.r)
			case TypeString:
				got = vm.string(reg.r)
			}
			if !reflect.DeepEqual(reg.value, got) {
				t.Errorf("source %q, register %s[%d]: expecting %#v (type %T), got %#v (type %T)", src, reg.typ, reg.r, reg.value, reg.value, got, got)
			}
		}
	}
}
