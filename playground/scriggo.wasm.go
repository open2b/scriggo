// +build js,wasm

package main

import (
	"syscall/js"

	"scriggo"
)

func main() {

	window := js.Global().Get("window")
	document := js.Global().Get("document")
	source := document.Call("getElementById", "Source")
	button := document.Call("getElementById", "Execute")

	button.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {

		main := scriggo.MapStringLoader{"main": source.Get("value").String()}

		program, err := scriggo.Load(main, 0)
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
