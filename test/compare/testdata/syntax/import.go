// errorcheck

import "time" // ERROR `expected 'package', found 'import'`

package main

import "time"
import "os"

const a = time.Second
const b = os.PathSeparator

import "time" // ERROR `syntax error: non-declaration statement outside function body`

func main() {
	import "time" // ERROR `syntax error: unexpected import, expecting }`
	switch {
		import "time" // ERROR `syntax error: unexpected import, expecting case or default or }`
	}
	if import "time" // ERROR `syntax error: unexpected import, expecting expression`
}
