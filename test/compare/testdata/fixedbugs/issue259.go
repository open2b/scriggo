// run

package main

import "fmt"

func test1() {
	switch interface{}(nil).(type) {
	default:
		panic("default")
	case int:
		panic("int")
	case nil:
		fmt.Println("test1: nil")
	case string:
		panic("string")
	}
}

func test2() {
	switch interface{}(nil).(type) {
	case nil:
		fmt.Println("test2: nil")
	}
}

func test3() {
	var i interface{}
	switch i.(type) {
	case nil:
		fmt.Println("test3: nil")
	default:
		panic("default")
	}
}

func test4() {
	var i interface{}
	switch i.(type) {
	case nil:
		fmt.Println("test4: i =", i)
	default:
		panic("default")
	}
}

// https://github.com/open2b/scriggo/issues/298
//
// func test5() {
//  var i interface{}
//  switch x := i.(type) {
//  case nil:
//      fmt.Println("test4: i =", i, ", x =", x)
//  default:
//      panic("default")
//  }
// }

func main() {
	test1()
	test2()
	test3()
	test4()
	//test5() // https://github.com/open2b/scriggo/issues/298
}
