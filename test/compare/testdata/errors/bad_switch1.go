// errorcheck

package main

func main() {
	switch false {
	case 0: println("zero!") // ERROR `invalid case 0 in switch on false (mismatched types int and bool)`
	}
}
