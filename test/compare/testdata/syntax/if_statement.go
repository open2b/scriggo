// errorcheck

package main

func f() bool { return true }

func main() {

	ch := make(chan int, 10)
	for i := 0; i < 10; i++ {
		ch <- 1
	}

	chb := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		chb <- true
	}

	var i, j int
	_, _ = i, j

	if for // ERROR `syntax error: unexpected for, expecting expression`
	if {} // ERROR `syntax error: missing condition in if statement`
	if ; {} // ERROR `syntax error: missing condition in if statement`
	if true {}
	if true for {} // ERROR `syntax error: unexpected for, expecting {`
	if true, true {} // ERROR `syntax error: unexpected {, expecting := or = or comma`
	if true, true for {} // ERROR `syntax error: unexpected for, expecting := or = or comma`
	if for {} // ERROR `syntax error: unexpected for, expecting expression`
	if ; true {}
	if ; {} // ERROR `syntax error: missing condition in if statement`
	if print(); true {}
	if <-ch; true {}
	if ch <- 3; true {}
	if i++; true {}
	if i--; true {}
	if i = 3; true {}
	if i = ; true {} // ERROR `unexpected semicolon, expecting expression`
	if i += 1; true {}
	if i, j = 3, 5; true {}
	if i := 3; i == 3 {}
	if i, j := 3, 5; i+j == 8 {}
	if print() {} // ERROR `print() used as value`
	if f() {}
	if <-ch {} // ERROR `non-bool <-ch (type int) used as if condition`
	if <-chb {}
	if ch <- 3 {} // ERROR `cannot use ch <- 3 as value`
	if i++ for {} // ERROR `syntax error: cannot use i++ as value`
	if i++ {} // ERROR `syntax error: cannot use i++ as value`
	if i-- {} // ERROR `syntax error: cannot use i-- as value`
	if i = 3 {} // ERROR `syntax error: cannot use assignment (i) = (3) as value`
	if a, b = 3, 6 {} // ERROR `syntax error: cannot use assignment (a, b) = (3, 6) as value`
	if i := 3 { _ = i } // ERROR `syntax error: cannot use i := 3 as value`
	if print(); {} // ERROR `syntax error: missing condition in if statement`
	if <-ch; {} // ERROR `syntax error: missing condition in if statement`
	if ch <- 3; {} // ERROR `syntax error: missing condition in if statement`
	if i++; {} // ERROR `syntax error: missing condition in if statement`
	if i = 3; {} // ERROR `syntax error: missing condition in if statement`
	if i += 1; {} // ERROR `syntax error: missing condition in if statement`
	if i := 3; { _ = i } // ERROR `syntax error: missing condition in if statement`

}