package scrigo

import (
	"bytes"
	"fmt"
	"testing"
)

var rendererCallFuncTests = []struct {
	src     string
	res     string
	globals packageNameScope
}{
	{"func f() {}; f()", "", packageNameScope{}},
	{"func f(x int) int { return x }; f(2)", "2", packageNameScope{}},
	{"func f(_ int) { }; f(2)", "", packageNameScope{}},
	{"func f(int) {}; f(1)", "", packageNameScope{}},
	{"func f(x, y int) int { return x + y }; (f(1, 2))", "3", packageNameScope{}},
	{"func f(_, _ int) { }; f(1, 2)", "", packageNameScope{}},
	{"func f(x int, y int) int { return x + y }; (f(1, 2))", "3", packageNameScope{}},
	{"func f(_ int, y int) int { return y }; (f(1, 2))", "2", packageNameScope{}},
	{"func f(...int) {}; f(1, 2, 3)", "", packageNameScope{}},
	{"func f(x ...int) int { s := 0; for _, i := range x { s += i }; return s }; (f(1, 2, 3))", "6", packageNameScope{}},
	{"func f(_ ...int) { }; f(1, 2, 3)", "", packageNameScope{}},
	// {"func f(x, y ...int) int { s := 0; for _, i := range y { s += i }; return s }; (f(1, 2, 3, 4))", "9", packageNameScope{}},
	// {"func f(_, _ ...int) { }; f(1, 2, 3, 4)", "", packageNameScope{}},
	// {"func f(_, y ...int) int { s := 0; for _, i := range y { s += i }; return s }; (f(1, 2, 3, 4))", "9", packageNameScope{}},
	// {"func f(x, _ ...int) int { return x }; (f(1, 2, 3, 4))", "1", packageNameScope{}},
	{"func f(x []int) int { s := 0; for _, i := range x { s += i }; return s }; (f([]int{1, 2, 3, 4}))", "10", packageNameScope{}},
	{"func f(x, y []int) int { s := 0; for _, i := range y { s += i }; return s }; (f([]int{1}, []int{2, 3, 4}))", "9", packageNameScope{}},
}

func TestRenderCallFunc(t *testing.T) {
	for _, stmt := range rendererCallFuncTests {
		r := bytes.NewReader([]byte(stmt.src))
		script, err := CompileScript(r, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		v, err := ExecuteScript(script, nil)
		if err != nil {
			t.Errorf("source: %q, %s\n", stmt.src, err)
			continue
		}
		res := ""
		if len(v) > 0 {
			res = fmt.Sprintf("%v", v[0])
		}
		if res != stmt.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", stmt.src, res, stmt.res)
		}
	}
}
