// errorcheck

package main

// TODO: note that the error should be: "syntax error: unexpected newline in type declaration"

type T // ERROR "^syntax error: unexpected semicolon in type declaration$"

func main() { }
