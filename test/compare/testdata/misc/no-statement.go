// errorcheck

:  // ERROR `expected 'package', found ':'`
∞  // ERROR `illegal character U+221E '∞'`

package main

:  // ERROR `non-declaration statement outside function body`

func main() {
	:  // ERROR `unexpected :, expected }`
	∞  // ERROR `illegal character U+221E '∞'`
}

:  // ERROR `non-declaration statement outside function body`
