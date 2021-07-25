// errorcheck

// This test file contains BOM runes.

package main

func main() {
 	_ = "a ﻿b" // ERROR `invalid BOM in the middle of the file`
	_ = `c﻿ d` // ERROR `invalid BOM in the middle of the file`
	_ = '﻿' // ERROR `invalid BOM in the middle of the file`
 	/* this is a ﻿ bom */ // ERROR `invalid BOM in the middle of the file`
 	// another ﻿ bom // ERROR `invalid BOM in the middle of the file`
	﻿ // another bom // ERROR `invalid BOM in the middle of the file`
}
