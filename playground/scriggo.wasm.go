//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

var packages native.Packages

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
			err := program.Run(nil)
			if cb.IsNull() || cb.IsUndefined() {
				return
			}
			if err != nil {
				msg := err.Error()
				if _, ok := err.(*scriggo.PanicError); ok {
					msg = "panic: " + msg
				}
				cb.Invoke(msg)
				return
			}
			cb.Invoke(nil)
		}()
		return nil
	})
	value["disassemble"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		asm, _ := program.Disassemble("main")
		return js.ValueOf(string(asm))
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

	Scriggo.Set("build", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var src string
		var cb js.Value
		if len(args) > 0 {
			src = args[0].String()
			if len(args) > 1 {
				cb = args[1]
			}
		}
		go func() {
			fsys := scriggo.Files{"main.go": []byte(src)}
			program, err := scriggo.Build(fsys, &scriggo.BuildOptions{AllowGoStmt: true, Packages: packages})
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
