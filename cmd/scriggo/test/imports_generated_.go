// Code generated by scriggo-generate, based on file "test/imports.go". DO NOT EDIT.
//+build

package test

import (
	fmt "fmt"
	"reflect"
	"scriggo"
)

func init() {
	test = scriggo.Packages{

		"fmt": {
			Name: "fmt",
			Declarations: map[string]interface{}{
				"Errorf":     fmt.Errorf,
				"Formatter":  reflect.TypeOf(new(fmt.Formatter)).Elem(),
				"Fprint":     fmt.Fprint,
				"Fprintf":    fmt.Fprintf,
				"Fprintln":   fmt.Fprintln,
				"Fscan":      fmt.Fscan,
				"Fscanf":     fmt.Fscanf,
				"Fscanln":    fmt.Fscanln,
				"GoStringer": reflect.TypeOf(new(fmt.GoStringer)).Elem(),
				"Print":      fmt.Print,
				"Printf":     fmt.Printf,
				"Println":    fmt.Println,
				"Scan":       fmt.Scan,
				"ScanState":  reflect.TypeOf(new(fmt.ScanState)).Elem(),
				"Scanf":      fmt.Scanf,
				"Scanln":     fmt.Scanln,
				"Scanner":    reflect.TypeOf(new(fmt.Scanner)).Elem(),
				"Sprint":     fmt.Sprint,
				"Sprintf":    fmt.Sprintf,
				"Sprintln":   fmt.Sprintln,
				"Sscan":      fmt.Sscan,
				"Sscanf":     fmt.Sscanf,
				"Sscanln":    fmt.Sscanln,
				"State":      reflect.TypeOf(new(fmt.State)).Elem(),
				"Stringer":   reflect.TypeOf(new(fmt.Stringer)).Elem(),
			},
		},
	}
}
