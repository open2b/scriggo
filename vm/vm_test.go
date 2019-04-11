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

// ptrTo returns a pointer to a copy of v.
func ptrTo(v interface{}) interface{} {
	rv := reflect.New(reflect.TypeOf(v))
	rv.Elem().Set(reflect.ValueOf(v))
	return rv.Interface()
}

var stmtTests = []struct {
	name      string
	src       string
	registers []reg
}{

	{
		"Simple assignment",
		`a := 10; _ = a`,
		[]reg{
			{TypeInt, 0, int64(10)}, // a
		},
	},
	// {
	// 	"Simple assignment with a 'big' int",
	// 	`
	// 	a := 1234
	// 	_ = a
	// 	`,
	// 	[]reg{
	// 		{TypeInt, 0, int64(1234)},
	// 	},
	// },
	{
		"Assignment with maths",
		`a := 5; b := 10; c := a + b; _ = c`,
		[]reg{
			{TypeInt, 0, int64(5)},  // a
			{TypeInt, 1, int64(10)}, // b
			{TypeInt, 2, int64(15)}, // c
		},
	},
	{
		"If else",
		`a := 10; c := 0; if a > 5 { c = 1 } else { c = 2 }; _ = c`,
		[]reg{
			{TypeInt, 0, int64(10)}, // a
			{TypeInt, 1, int64(1)},  // c
		},
	},
	{
		"If else",
		`a := 10; c := 0; if a <= 5 { c = 1 } else { c = 2 }; _ = c`,
		[]reg{
			{TypeInt, 0, int64(10)}, // a
			{TypeInt, 1, int64(2)},  // c
		},
	},
	{
		"Maths",
		`a := 10 + 20 - 4; _ = a`,
		[]reg{
			{TypeInt, 0, int64(26)}, // a
		},
	},
	{
		"If with init",
		`c := 0; if x := 1; x == 1 { c = 1 } else { c = 2 }; _ = c`,
		[]reg{
			{TypeInt, 0, int64(1)}, // c
		},
	},
	{
		"String concatenation",
		`a := "s"; _ = a; b := "ee" + "ff"; _ = b`,
		[]reg{
			{TypeString, 0, "s"},    // a
			{TypeString, 2, "eeff"}, // b
		},
	},
	{
		"Empty int slice",
		`a := []int{}; _ = a`,
		[]reg{
			{TypeIface, 0, []int{}}, // a
		},
	},
	// `a := new(int); _ = a`,
	// reg{
	// 	{TypeIface, 0, ptrTo(int64(0))},
	// },
	{
		"Empty slice slice",
		`a := []string{}; _ = a`,
		[]reg{
			{TypeIface, 0, []string{}}, // a
		},
	},
	{
		"Empty byte slice",
		`a := []int{}; b := []byte{}; _ = a; _ = b`,
		[]reg{
			{TypeIface, 0, []int{}},  // a
			{TypeIface, 1, []byte{}}, // b
		},
	},
	// {
	// 	"Int slice composite literal",
	// 	`
	// 	a := []int{1,2,4}
	// 	_ = a
	// 	`,
	// 	[]reg{
	// 		{TypeIface, 0, []int{1, 2, 5}},
	// 	},
	// },
	{
		"Builtin len (with a constant argument)",
		`a := len("abc"); _ = a`,
		[]reg{
			{TypeInt, 0, int64(3)}, // a
		},
	},
	{
		"Switch",
		`
		a := 0
		switch 1 + 1 {
		case 1:
			a = 10
		case 2:
			a = 20
		case 3:
			a = 30
		}
		_ = a
		`,
		[]reg{
			{TypeInt, 0, int64(20)},
		},
	},
	{
		"Switch with fallthrough",
		`
		a := 0
		switch 1 + 1 {
		case 1:
			a = 10
		case 2:
			a = 20
			fallthrough
		case 3:
			a = 30
		}
		_ = a
		`,
		[]reg{
			{TypeInt, 0, int64(30)},
		},
	},
	{
		"Switch with default",
		`
			a := 0
			switch 10 + 10 {
			case 1:
				a = 10
			case 2:
				a = 20
			default:
				a = 80
			case 3:
				a = 30
			}
			_ = a
		`,
		[]reg{
			{TypeInt, 0, int64(80)},
		},
	},
	{
		"Switch with default and fallthrough",
		`
		a := 0
		switch 2 + 2 {
		case 1:
			a = 10
		default:
			a = 80
			fallthrough
		case 2:
			a = 1
			fallthrough
		case 3:
			a = 30
		case 40:
			a = 3
		}
		_ = a
		`,
		[]reg{
			{TypeInt, 0, int64(30)},
		},
	},
}

func TestVM(t *testing.T) {
	DebugTraceExecution = testing.Verbose()
	for _, cas := range stmtTests {
		t.Run(cas.name, func(t *testing.T) {
			src := cas.src
			registers := cas.registers
			fullSrc := "package main\nfunc main(){\n" + src + "\n}\n"
			r := parser.MapReader{"/test.go": []byte(fullSrc)}
			comp := NewCompiler(r, nil)
			pkg, err := comp.Compile("/test.go")
			if err != nil {
				t.Errorf("source: %q, compiler error: %s", src, err)
				return
			}
			vm := New(pkg)
			_, err = vm.Run("main")
			if err != nil {
				t.Errorf("source: %q, execution error: %s", src, err)
				return
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
		})
	}
}
