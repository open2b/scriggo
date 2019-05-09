// +build js,wasm

package main

import (
	"strings"
	"syscall/js"

	"scrigo"
	"scrigo/compiler/parser"
)

func main() {

	window := js.Global().Get("window")
	document := js.Global().Get("document")
	source := document.Call("getElementById", "Source")
	button := document.Call("getElementById", "Execute")

	button.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		r := parser.MapReader{}
		comp := scrigo.NewCompiler(r, nil)
		program, err := comp.Compile(strings.NewReader(source.Get("value").String()))
		if err != nil {
			window.Call("alert", err.Error())
			return nil
		}
		err = scrigo.Execute(program)
		if err != nil {
			window.Call("alert", err.Error())
			return nil
		}

		print("ok")

		return nil
	}))

	select {}
}
