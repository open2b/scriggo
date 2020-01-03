// +build js,wasm

package main

import (
	"strings"
	"syscall/js"

	"scriggo"
)

func main() {

	window := js.Global().Get("window")
	document := js.Global().Get("document")
	source := document.Call("getElementById", "Source")
	button := document.Call("getElementById", "Execute")

	button.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		src := strings.NewReader(source.Get("value").String())
		program, err := scriggo.Load(src, nil, nil)
		if err != nil {
			window.Call("alert", err.Error())
			return nil
		}
		_, err = program.Run(nil)
		if err != nil {
			window.Call("alert", err.Error())
			return nil
		}

		print("ok")

		return nil
	}))

	select {}
}
