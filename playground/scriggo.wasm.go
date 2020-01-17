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
		var cb js.Value
		if len(args) > 0 {
			cb = args[0]
		}
		go func() {
			_, err := program.Run(nil)
			if cb.IsNull() || cb.IsUndefined() {
				return
			}
			if err != nil {
				cb.Invoke(err.Error())
				return
			}
			cb.Invoke(nil)
		}()
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
		value["release"].(js.Func).Release()
		return nil
	})
	return Program{Value: js.ValueOf(value)}
}

func main() {

	Scriggo := js.ValueOf(map[string]interface{}{})

	Scriggo.Set("load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var src string
		var cb js.Value
		if len(args) > 0 {
			src = args[0].String()
			if len(args) > 1 {
				cb = args[1]
			}
		}
		go func() {
			program, err := scriggo.Load(strings.NewReader(src), packages, nil)
			if cb.IsNull() || cb.IsUndefined() {
				return
			}
			if err != nil {
				cb.Invoke(nil, err.Error())
				return
			}
			cb.Invoke(newProgram(program), nil)
		}()
		return nil
	}))

	window := js.Global().Get("window")
	window.Set("Scriggo", Scriggo)

	select {}
}
