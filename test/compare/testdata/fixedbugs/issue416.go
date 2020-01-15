// compile

package main

func main() {

	_ = (func())(nil)

	_ = (func())(func(){})

	switch interface{}(0).(type) {
	case func():

	}

}
