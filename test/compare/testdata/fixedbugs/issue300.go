// errorcheck

package main

func main() {
	type ( T1 int , T2 int ) // ERROR `syntax error: unexpected comma, expecting semicolon or newline or )`
	type ( T1 int T2 int ) // ERROR `syntax error: unexpected T2, expecting semicolon or newline or )`
}
