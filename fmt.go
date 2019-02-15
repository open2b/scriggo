// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"fmt"
	"reflect"
)

var _fmt = Package{
	"Errorf":     fmt.Errorf,
	"Formatter":  reflect.TypeOf((*fmt.Formatter)(nil)).Elem(),
	"Fprint":     fmt.Fprint,
	"Fprintf":    fmt.Fprintf,
	"Fprintln":   fmt.Fprintln,
	"Fscan":      fmt.Fscan,
	"Fscanf":     fmt.Fscanf,
	"Fscanln":    fmt.Fscanln,
	"GoStringer": reflect.TypeOf((*fmt.GoStringer)(nil)).Elem(),
	"Print":      fmt.Print,
	"Printf":     fmt.Printf,
	"Println":    fmt.Println,
	"Scan":       fmt.Scan,
	"ScanState":  reflect.TypeOf((*fmt.ScanState)(nil)).Elem(),
	"Scanf":      fmt.Scanf,
	"Scanln":     fmt.Scanln,
	"Scanner":    reflect.TypeOf((*fmt.Scanner)(nil)).Elem(),
	"Sprint":     fmt.Sprint,
	"Sprintf":    fmt.Sprintf,
	"Sprintln":   fmt.Sprintln,
	"Sscan":      fmt.Sscan,
	"Sscanf":     fmt.Sscanf,
	"Sscanln":    fmt.Sscanln,
	"State":      reflect.TypeOf((*fmt.State)(nil)).Elem(),
	"Stringer":   reflect.TypeOf((*fmt.Stringer)(nil)).Elem(),
}
