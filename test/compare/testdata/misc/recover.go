// skip

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
	test15()
	test16()
	test17()
	test18()
	test19()
	test20()

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
	f14b := func() {
		panic(1)
	}
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	f14b()
}

func test15() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	func() {
		panic(1)
	}()
}

func test16() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	defer func() {
		func() {
			defer func() {
				v := recover()
				expectRecover(v, 2)
			}()
			panic(2)
		}()
	}()
	panic(1)
}

func test17() {
	defer func() {
		v := recover()
		notExpectRecover(v)
	}()
	defer func() {
		defer recover()
		defer func() {
			v := recover()
			expectRecover(v, 2)
		}()
		panic(2)
	}()
	panic(1)
}

func test18() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	defer func() {
		defer func() {
			v := recover()
			expectRecover(v, 2)
		}()
		defer recover() // no-op
		panic(2)
	}()
	panic(1)
}

func test19() {
	defer func() {
		v := recover()
		expectRecover(v, 1)
	}()
	runtime.GOMAXPROCS(1)
	go recover()
	runtime.Gosched()
	panic(1)
}

func test20() {
	defer func() {
		got := recover()
		if got != nil {
			log.Printf("expected recover nil, got %#v", got)
			os.Exit(-1)
		}
	}()
	panic(nil)
}
