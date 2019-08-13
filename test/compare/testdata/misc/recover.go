// run

package main

import (
	"log"
	"os"
	"reflect"
	"runtime"
)

func main() {

	test1()
	test2()
	test3()
	test4()
	test5()
	test6()
	test7()
	test8()
	test9()
	test10()
	test11()
	test12()
	test13()
	test14()

}

func expectRecover(got, expected interface{}) {
	if got == nil {
		log.Printf("expected recover %#v, got nil", expected)
		os.Exit(-1)
	}
	if !reflect.DeepEqual(got, expected) {
		log.Printf("expected recover %#v, got %T %#v", expected, got, got)
		os.Exit(-1)
	}
}

func expectRuntimeRecover(got interface{}, expected string) {
	if got == nil {
		log.Printf("expected recover %#v, got nil", expected)
		os.Exit(-1)
	}
	e, ok := got.(runtime.Error)
	if !ok {
		log.Printf("expected recover runtime error, got %T %#v", got, got)
		os.Exit(-1)
	}
	if e.Error() != "runtime error: "+expected {
		log.Printf("expected recover %q, got %q", expected, got)
		os.Exit(-1)
	}
}

func notExpectRecover(got interface{}) {
	if got != nil {
		log.Printf("not expected recover %#v", got)
		os.Exit(-1)
	}
}

func test1() {
	v := recover()
	notExpectRecover(v)
}

func test2() {
	defer func() {}()
	v := recover()
	notExpectRecover(v)
}

func test3() {
	defer func() {
		v := recover()
		notExpectRecover(v)
	}()
}

func test4() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	panic(1)
}

func test5() {
	defer func() {
		defer func() {

		}()
		v := recover()
		expectRecover(v, 1)
	}()
	panic(1)
}

func test6() {
	defer func() {
		defer func() {
			v := recover()
			notExpectRecover(v)
		}()
		v := recover()
		expectRecover(v, 1)
	}()
	panic(1)
}

func test7() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	defer func() {
		defer func() {
			v := recover()
			notExpectRecover(v)
		}()
	}()
	panic(1)
}

func test8() {
	defer func() {
		v := recover()
		expectRecover(v, 2)
	}()
	defer func() {
		panic(2)
	}()
	panic(1)
}

func test9() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	defer func() {
		defer func() {
			v := recover()
			expectRecover(v, 2)
		}()
		panic(2)
	}()
	panic(1)
}

func test10() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	test10b()
}

func test10b() {
	panic(1)
}

func test11() {
	defer func() {
		v := recover()
		expectRecover(v, 2)
	}()
	test11b()
}

func test11b() {
	defer func() {
		panic(2)
	}()
	panic(1)
}

func test12() {
	defer func() {
		v := recover()
		notExpectRecover(v)
	}()
	defer f12b()
	panic(1)
}

func f12b() {
	v := recover()
	expectRecover(v, 1)
}

func test13() {
	f13b := func() {
		v := recover()
		expectRecover(v, 1)
	}
	defer func() {
		v := recover()
		notExpectRecover(v)
	}()
	defer f13b()
	panic(1)
}

func test14() {
	defer func() {
		v := recover()
		expectRuntimeRecover(v, "index out of range")
	}()
	var a []int
	_ = a[0]
}
