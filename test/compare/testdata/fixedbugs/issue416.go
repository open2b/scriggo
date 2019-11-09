// compile

package main

func main() {

	// https://github.com/open2b/scriggo/issues/416
	// _ = (func())(nil)  // panic the emitter

	_ = (func())(func(){})

	switch interface{}(0).(type) {
	case func():

	}

}
