// run
//+build go1.17

package main

import (
	"log"
	"os"
	"reflect"
	"runtime"
)

func main() {
	test19()
}

// keep in sync with the same function in the "runtime_panic.go" test.
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
	if _, ok := os.LookupEnv("SCRIGGO"); ok {
		typ := reflect.TypeOf(e)
		if typ.PkgPath() != "github.com/open2b/scriggo/internal/runtime" {
			if typ.Name() == "TypeAssertionError" {
				log.Printf("expected type github.com/open2b/scriggo/internal/runtime.TypeAssertionError, got %s.TypeAssertionError for error %q", typ.PkgPath(), err)
			} else {
				log.Printf("expected type github.com/open2b/scriggo/internal/runtime.runtimeError, got %s.%s for error %q", typ.PkgPath(), typ.Name(), err)
			}
			os.Exit(1)
		}
	}
}

func test19() {
	defer recoverRuntimePanic("runtime error: cannot convert slice with length 1 to pointer to array with length 2")
	s := []int{1}
	_ = (*[2]int)(s)
}
