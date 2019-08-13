// run

package main

import (
	"log"
	"os"
	"runtime"

	"testpkg"
)

func main() {

	// test1() // expecting "runtime error: hash of unhashable type func()", got nothing
	// test2() // emitter panic: "TODO(Gianluca): not implemented"
	// test3() // expected "runtime error: invalid memory address or nil pointer dereference", got *reflect.ValueError &reflect.ValueError{Method:"reflect.Value.MethodByName", Kind:0x0}
	// test4() // expected "runtime error: index out of range", got "reflect: slice index out of range"
	test4b()
	test5()
	test5b()
	test6()
	test7()
	// test8() // expected "runtime error: slice bounds out of range", got "reflect.Value.Slice: slice index out of bounds"
	// test9() // expected "runtime error: slice bounds out of range", got "reflect.Value.Slice: slice index out of bounds"
	// test10() // expected "runtime error: slice bounds out of range", got <*reflect.ValueError> &reflect.ValueError{Method:"reflect.Value.Len", Kind:0x16}
	// test11() // expected "interface conversion: interface {} is int, not string", got nothing
	// test12() // expected "runtime error: invalid memory address or nil pointer dereference", got "reflect.Value.Call: call of nil function"
	test13()
	test14()
	test15()
	// test16() // emitter panic: "TODO(Gianluca): not implemented"
	test17()
	test18()
	test19()
	// test20() // expected "runtime error: makeslice: len out of range", got "reflect.MakeSlice: negative len"
	// test21() // expected "runtime error: makeslice: cap out of range", got "reflect.MakeSlice: len > cap"

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
	t.M()
}

func test4() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a [1]int
	var b = 2
	_ = a[b]
}

func test4b() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a [1]int
	var b = 2
	a[b] = 3
}

func test5() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a []int
	_ = a[0]
}

func test5b() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a []int
	a[0] = 1
}

func test6() {
	defer recoverRuntimePanic("runtime error: index out of range")
	var a string
	_ = a[0]
}

func test7() {
	defer recoverRuntimePanic("assignment to entry in nil map")
	var a map[string]string
	a["b"] = "c"
}

func test8() {
	defer recoverRuntimePanic("runtime error: slice bounds out of range")
	a := make([]int, 0)
	_ = a[1:]
}

func test9() {
	defer recoverRuntimePanic("runtime error: slice bounds out of range")
	a := [1]int{}
	b := 2
	_ = a[b:]
}

func test10() {
	defer recoverRuntimePanic("runtime error: slice bounds out of range")
	a := ""
	_ = a[1:]
}

func test11() {
	defer recoverRuntimePanic("interface conversion: interface {} is int, not string")
	var a interface{} = 5
	_ = a.(string)
}

func test12() {
	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
	var f func()
	f()
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
	var a *int
	_ = *a
}

//func test16() {
//	defer recoverRuntimePanic("runtime error: invalid memory address or nil pointer dereference")
//	var a *int
//	_ = &*a
//}

func test17() {
	defer recoverRuntimePanic("send on closed channel")
	a := make(chan int)
	close(a)
	a <- 3
}

func test18() {
	defer recoverRuntimePanic("close of closed channel")
	a := make(chan int)
	close(a)
	close(a)
}

func test19() {
	defer recoverRuntimePanic("close of nil channel")
	var a chan int
	close(a)
}

func test20() {
	defer recoverRuntimePanic("runtime error: makeslice: len out of range")
	a := -1
	_ = make([]int, a)
}

func test21() {
	defer recoverRuntimePanic("runtime error: makeslice: cap out of range")
	a := 1
	_ = make([]int, a+1, a)
}
