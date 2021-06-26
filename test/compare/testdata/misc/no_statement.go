// errorcheck

:  // ERROR `expected 'package', found ':'`
∞  // ERROR `invalid character U+221E '∞'`
}  // ERROR `expected 'package', found '}'`
$  // ERROR `invalid character U+0024 '$'`

package main

:  // ERROR `non-declaration statement outside function body`
}  // ERROR `non-declaration statement outside function body`

func main() {
	:  // ERROR `unexpected :, expecting }`
	∞  // ERROR `invalid character U+221E '∞'`
}

:  // ERROR `non-declaration statement outside function body`
}  // ERROR `non-declaration statement outside function body`

return // ERROR `non-declaration statement outside function body`
