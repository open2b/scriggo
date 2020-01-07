// errorcheck

package main

type T int

func main() {
	_ = T._ // ERROR "^T\._ undefined \(type T has no method _\)$"
}
