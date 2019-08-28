// skip : this test fails because the receive operation converts the type to
// the internal value, but this is wrong: if the channel is an chan interface,
// the value must be kept as is

// run

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test zero length structs.
// Used to not be evaluated.
// Issue 2232.

package main

func recv(c chan interface{}) struct{} {
	return (<-c).(struct{})
}

var m = make(map[interface{}]int)

func recv1(c chan interface{}) {
	defer rec()
	m[(<-c).(struct{})] = 0
}

func rec() {
	recover()
}

func main() {
	c := make(chan interface{})
	go recv(c)
	c <- struct{}{}
	go recv1(c)
	c <- struct{}{}
}
