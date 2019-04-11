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

var stmt_tests = map[string][]reg{

	`a := 10; _ = a`: []reg{
		{TypeInt, 0, int64(10)}, // a
	},
	// `a := 1234; _ = a`: []reg{
	// 	{TypeInt, 0, int64(1234)}, // a
	// },
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
	`a := "s"; _ = a; b := "ee" + "ff"; _ = b`: []reg{
		{TypeString, 0, "s"},    // a
		{TypeString, 2, "eeff"}, // b
	},
	`a := []int{}; _ = a`: []reg{
		{TypeIface, 0, []int{}}, // a
	},
	`a := []string{}; _ = a`: []reg{
		{TypeIface, 0, []string{}}, // a
	},
	`a := []int{}; b := []byte{}; _ = a; _ = b`: []reg{
		{TypeIface, 0, []int{}},  // a
		{TypeIface, 1, []byte{}}, // b
	},
	// `a := []int{1,2,4}; _ = a`: []reg{
	// 	{TypeIface, 0, []int{1, 2, 5}},
	// },
	`a := len("abc"); _ = a`: []reg{
		{TypeInt, 0, int64(3)}, // a
	},
	`a := 0; switch 1 + 1 { case 1: a = 10 ; case 2: a = 20; case 3: a = 30 }; _ = a`: []reg{
		{TypeInt, 0, int64(20)},
	},
	`a := 0; switch 1 + 1 { case 1: a = 10 ; case 2: a = 20; fallthrough; case 3: a = 30 }; _ = a`: []reg{
		{TypeInt, 0, int64(30)},
	},
	// `a := 0; switch 2 + 2 { case 1: a = 10 ; default: a = 80; case 2: a = 20; case 3: a = 30 }; _ = a`: []reg{
	// 	{TypeInt, 0, int64(80)},
	// },
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for src, registers := range stmt_tests {
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
				got = vm.general(reg.r)
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
