// skip : expecting 'main.Iface', got 'Iface'

// paniccheck

package main

type Iface interface{}

func main() {
	var i Iface
	_ = i.(Iface)
}
