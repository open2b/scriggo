// errorcheck

package main

func main() {
	type _ = *  // ERROR `unexpected }, expecting type`
	_ = a +     // ERROR `unexpected }, expecting expression`
}