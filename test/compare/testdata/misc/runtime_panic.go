// run

package main

import (
	"log"
	"os"
	"runtime"
	"strings"

	"testpkg"
)

func main() {

	isGo112 := strings.HasPrefix(runtime.Version(), "go1.12")

	test1()
	// test2() // emitter panic: "TODO(Gianluca): not implemented"
	test3()
	test3b()
	if isGo112 {
		test4()
		test4b()
		test5()
		test5b()
		test6()
	}
	test7()
	if isGo112 {
		test8()
		test9()
		test10()
	}
	// test11() // expected "interface conversion: interface {} is int, not testpkg.S", got nothing
	test12()
	test12b()
	test12c()
	test12d()
	test12e()
	test12f()
	test13()
	test14()
	// test15() // expected "runtime error: invalid memory address or nil pointer dereference", got &reflect.ValueError{Method:"reflect.Value.Uint", Kind:0x0}
	// test16() // emitter panic: "TODO(Gianluca): not implemented"
	test17()
	test18()
	test19()
	test20()
	test21()
	test22()

}

func recoverRuntimePanic(err string) {
	v := recover()
	if v == nil {
		log.Printf("expected error %q, got nil", err)
		os.Exit(1)
	}
	e, ok := v.(runtime.Error)
	if !ok {
		log.Printf("expected error %q, got %#v (type %T)", err, v, v)
		os.Exit(1)
	}
	if e.Error() != err {
		log.Printf("expected error %q, got %q", err, e)
		os.Exit(1)
	}
}

func test1() {
	defer recoverRuntimePanic("runtime error: hash of unhashable type func()")
	a := map[interface{}]string{}
	a[func() {}] = "c"
}

//func test2() {
//	defer recoverRuntimePanic("runtime error: hash of unhashable type func()")
//	tp := &testpkg.TestPointInt{}
//	tp.A = 5
//}

func test3() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var t testpkg.I
	_ = t.M
}

func test3b() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var t testpkg.I
	t.M()
}

func test4() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a [1]testpkg.T
	var b = 2
	_ = a[b]
}

func test4b() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a [1]testpkg.T
	var b = 2
	a[b] = 3
}

func test5() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a []testpkg.T
	_ = a[0]
}

func test5b() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a []testpkg.T
	a[0] = 1
}

func test6() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a testpkg.S
	_ = a[0]
}

func test7() {
	defer recoverRuntimePanic("assignment to entry in nil map")
	var a map[string]string
	a["b"] = "c"
}

func test8() {
	defer recoverRuntimePanic("runtime error: slice bounds out of range")
	a := make([]testpkg.T, 0)
	_ = a[1:]
}

func test9() {
	defer recoverRuntimePanic("runtime error: slice bounds out of range")
	a := [1]testpkg.T{}
	b := 2
	_ = a[b:]
}

func test10() {
	defer recoverRuntimePanic("runtime error: slice bounds out of range")
	a := testpkg.S("")
	_ = a[1:]
}

func test11() {
	defer recoverRuntimePanic("interface conversion: interface {} is int, not testpkg.S")
	var a interface{} = 5
	_ = a.(testpkg.S)
}

func test12() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var f func()
	f()
}

func test12b() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var f func(string) int
	f("")
}

func test12c() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var f func(a ...testpkg.T)
	f()
}

var f1 func()
var f2 func(string) int
var f3 func(a ...testpkg.T)

func test12d() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	f1()
}

func test12e() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	f2("")
}

func test12f() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	f3()
}

func test13() {
	defer recoverRuntimePanic("runtime error: integer divide by zero")
	var a = 0
	_ = 1 / a
}

func test14() {
	defer recoverRuntimePanic("runtime error: comparing uncomparable type []int")
	var a interface{} = []int{0}
	var b interface{} = []int{0}
	_ = a == b
}

func test15() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var a *testpkg.T
	_ = *a
}

//func test16() {
//	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
//	var a *int
//	_ = &*a
//}

func test17() {
	defer recoverRuntimePanic("send on closed channel")
	a := make(chan testpkg.T)
	close(a)
	a <- 3
}

func test18() {
	defer recoverRuntimePanic("close of closed channel")
	a := make(chan testpkg.T)
	close(a)
	close(a)
}

func test19() {
	defer recoverRuntimePanic("close of nil channel")
	var a chan testpkg.T
	close(a)
}

func test20() {
	defer recoverRuntimePanic("runtime error: makeslice: len out of range")
	a := -1
	_ = make([]testpkg.T, a)
}

func test21() {
	defer recoverRuntimePanic("runtime error: makeslice: cap out of range")
	a := -1
	_ = make([]testpkg.T, 0, a)
}

func test22() {
	defer recoverRuntimePanic("runtime error: makeslice: cap out of range")
	a := 1
	_ = make([]testpkg.T, a+1, a)
}
