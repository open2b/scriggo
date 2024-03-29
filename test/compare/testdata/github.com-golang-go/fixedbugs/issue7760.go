// skip : import "unsafe" https://github.com/open2b/scriggo/issues/288

// errorcheck

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Verify that pointers can't be used as constants.

package main

import "unsafe"

type myPointer unsafe.Pointer

const _ = unsafe.Pointer(uintptr(1)) // ERROR "is not (a )?constant"
const _ = myPointer(uintptr(1)) // ERROR "is not (a )?constant"

const _ = (*int)(unsafe.Pointer(uintptr(1))) // ERROR "is not (a )?constant"
const _ = (*int)(myPointer(uintptr(1))) // ERROR "is not (a )?constant"

const _ = uintptr(unsafe.Pointer(uintptr(1))) // ERROR "is not (a )?constant"
const _ = uintptr(myPointer(uintptr(1))) // ERROR "is not (a )?constant"

const _ = []byte("") // ERROR "is not (a )?constant"
const _ = []rune("") // ERROR "is not (a )?constant"
