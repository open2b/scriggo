// errorcheck

package main

var ( v int // ERROR `unexpected func, expecting name`
var ( v int; // ERROR `unexpected func, expecting name`
var ( v int + // ERROR `unexpected +, expecting semicolon or newline or )`

const ( c int = 0 // ERROR `unexpected func, expecting name`
const ( c int = 0; // ERROR `unexpected func, expecting name`
const ( c = iota; c2 // ERROR `unexpected func, expecting name`

func main() {

}
