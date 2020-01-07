// errorcheck

package main

func main() {
	v += // ERROR `syntax error: unexpected }, expecting expression`
}