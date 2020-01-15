// errorcheck

package main

func main() {

	var i, v int
	_, _ = i, v

	ch := make(chan int, 10)
	for i := 0; i < 10; i++ {
		ch <- 1
	}

	chb := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		chb <- true
	}

	for if // ERROR `syntax error: unexpected if, expecting expression`
	for { break }
	for ; {} // ERROR `syntax error: unexpected {, expecting for loop condition`
	for ; true {} // ERROR `syntax error: unexpected {, expecting semicolon or newline`
	for ; ; { break }
	for a := 5 {} // ERROR `syntax error: a := 5 used as value`
	for a := 5; {} // ERROR `syntax error: unexpected {, expecting for loop condition`
	for a := 5; true {} // ERROR `syntax error: unexpected {, expecting semicolon or newline`
	for a := false; a; {}
	for i := 0; i < 10; i++ {}
	for i, j := 0, 10; i < j; i++ {}
	for i, j := 0, 10; i < j; i, j = i+1, j {}
	for ; <- chb; { break }
	for true; ; {} // ERROR `true evaluated but not used`
	for ; ; true {} // ERROR `true evaluated but not used`
	for ; ; i := 0 {} // ERROR `syntax error: cannot declare in post statement of for loop`
	for ; ; i = 0 { break }
	for ; ; i++ { break }
	for false {}
	for 1 + 2 == 5 {}
	for range []int(nil) {}
	for i := range []int(nil) { _ = i }
	for i = range []int(nil) {}
	for i, v range []int(nil) {} // ERROR `unexpected range, expecting := or = or comma`
	for i, v = range []int(nil) {}
	for i, v := range []int(nil) { _, _ = i, v }
	for i, v = 1, 2 range []int(nil) {} // ERROR `syntax error: unexpected range, expecting {`
	for a range []int(nil) {} // ERROR `syntax error: unexpected range, expecting {`
	for range {} // ERROR `syntax error: unexpected {, expecting expression`
	for range for {} // ERROR `syntax error: unexpected for, expecting expression`
}