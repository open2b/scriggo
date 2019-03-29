package parser

import (
	"reflect"
	"scrigo/ast"
	"testing"
)

func TestCheckScript(t *testing.T) {
	cases := []struct {
		src  string
		main *GoPackage
	}{
		{
			src: `
				s := SliceInt{1,2,3}
				println(len(s))
				a := 20
			`,
			main: &GoPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"SliceInt": reflect.SliceOf(reflect.TypeOf(int(0))),
				},
			},
		},
	}
	for _, c := range cases {
		tree, err := ParseSource([]byte(c.src), ast.ContextNone)
		if err != nil {
			t.Errorf("parsing error: %s", err)
			continue
		}
		_, err = checkScript(tree, c.main)
		if err != nil {
			t.Errorf("type checking error: %s", err)
		}
	}
}
