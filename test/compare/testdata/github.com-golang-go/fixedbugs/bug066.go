// skip : investigate on this

// compile

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type Scope = struct {
	Entries map[string] *Object;
}


type Type = struct {
	Scope *Scope;
}


type Object struct {
	Typ *Type;
}


func Lookup(scope *Scope) *Object {
	return scope.Entries["foo"];
}

func main() { }