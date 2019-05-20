package script

import (
	"reflect"
	"testing"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/native"
)

func TestCheckScript(t *testing.T) {
	cases := []struct {
		src  string
		main *native.GoPackage
	}{
		{
			src: `
				s := SliceInt{1,2,3}
				println(len(s))
				a := 20
			`,
			main: &native.GoPackage{
				Name: "main",
				Declarations: map[string]interface{}{
					"SliceInt": reflect.SliceOf(reflect.TypeOf(int(0))),
				},
			},
		},
	}
	for _, c := range cases {
		tree, err := compiler.ParseSource([]byte(c.src), ast.ContextNone)
		if err != nil {
			t.Errorf("parsing error: %s", err)
			continue
		}
		_, err = typecheck(tree, c.main)
		if err != nil {
			t.Errorf("type checking error: %s", err)
		}

	}
}
