// errorcheck

package main

func main() {
	switch i := 0; i.(type) { // ERROR `cannot type switch on non-interface value i (type int)`
	}
}
