// errorcheck

:  // ERROR `expected 'package', found ':'`

package main

:  // ERROR `non-declaration statement outside function body`

func main() {
	:  // ERROR `unexpected :, expected }`
}

:  // ERROR `non-declaration statement outside function body`
