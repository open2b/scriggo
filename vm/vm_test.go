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

var cases = []struct {
	src      string
	expected []reg
}{

	{"a := 10; _ = a",
		[]reg{
			{TypeInt, 0, int64(10)},
		}},

	{"a := 5; b := 10; c := a + b; _ = c",
		[]reg{
			{TypeInt, 0, int64(5)},
			{TypeInt, 1, int64(10)},
			{TypeInt, 2, int64(15)},
		}},
}

func TestVM(t *testing.T) {
	DebugTraceExecution = false
	for _, c := range cases {
		fullSrc := "package main\nfunc main(){\n" + c.src + "\n}\n"
		r := parser.MapReader{"/test.go": []byte(fullSrc)}
		comp := NewCompiler(r, nil)
		pkg, err := comp.Compile("/test.go")
		if err != nil {
			t.Errorf("source: %q, compiler error: %s", c.src, err)
			continue
		}
		vm := New(pkg)
		_, err = vm.Run("main")
		if err != nil {
			t.Errorf("source: %q, execution error: %s", c.src, err)
			continue
		}
		for _, reg := range c.expected {
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
				t.Errorf("source %q, register %s[%d]: expecting %v (type %T), got %v (type %T)", c.src, reg.typ, reg.r, reg.value, reg.value, got, got)
			}
		}
	}
}
