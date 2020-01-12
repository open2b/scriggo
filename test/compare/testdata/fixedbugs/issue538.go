// errorcheck

package main

func main() {

	for i := 0; i < 10; i = { } // ERROR `syntax error: unexpected {, expecting expression`

	select { case i = : } // ERROR `syntax error: unexpected :, expecting expression`

	if i = ; true { } // ERROR `syntax error: unexpected semicolon, expecting expression`

	i = // ERROR `syntax error: unexpected }, expecting expression`

	switch i = ; {} // ERROR `syntax error: unexpected semicolon, expecting expression`

	switch x := 2; x = {}  // ERROR `syntax error: unexpected {, expecting expression`

}