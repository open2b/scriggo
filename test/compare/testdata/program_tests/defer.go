// run

package main

import "fmt"

func mark(m string) {
	fmt.Print(m + ",")
}

func f() {
	mark("f")
	defer func() {
		mark("f.1")
		defer func() {

		}()
	}()
	defer func() {
		mark("f.2")
		defer func() {
			mark("f.2.1")
		}()
		h()
	}()
	g()
	mark("ret f")
}

func g() {
	mark("g")
}

func h() {
	mark("h")
}

func main() {
	mark("main")
	defer func() {
		mark("main.1")
		h()
		mark("main.1")
	}()
	f()
	mark("ret main")
}
