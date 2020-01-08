// +build js,wasm

package main

import (
	"strings"
	"syscall/js"

	"scriggo"
)

var packages scriggo.Packages

type Program struct {
	js.Value
}

func newProgram(program *scriggo.Program) Program {
	value := map[string]interface{}{}
	value["run"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		_, err := program.Run(nil)
		if err != nil {
			return err.Error()
		}
		return nil
	})
	value["disassemble"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		b := strings.Builder{}
		_, _ = program.Disassemble(&b, "main")
		return js.ValueOf(b.String())
	})
	value["release"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value["run"].(js.Func).Release()
		value["disassemble"].(js.Func).Release()
		return nil
	})
	return Program{Value: js.ValueOf(value)}
}

func main() {

	Scriggo := js.ValueOf(map[string]interface{}{})

	Scriggo.Set("load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var src string
		if len(args) > 0 {
			src = args[0].String()
		}
		program, err := scriggo.Load(strings.NewReader(src), packages, nil)
		if len(args) == 1 {
			return nil
		}
		cb := args[1]
		if err != nil {
			cb.Invoke(nil, err.Error());
			return nil
		}
		cb.Invoke(newProgram(program), nil);
		return nil
	}))

	window := js.Global().Get("window")
	window.Set("Scriggo", Scriggo)

	select {}
}
