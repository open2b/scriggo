// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

func FuncOf(in, out []reflect.Type, variadic bool) reflect.Type {

	// First: check if this function contains a Scriggo type in its parameters.
	// If not, such function can be created with reflect.FuncOf without any
	// problem.
	isScriggoType := false
	for _, t := range in {
		if _, ok := t.(ScriggoType); ok {
			isScriggoType = true
			break
		}
	}
	for _, t := range out {
		if _, ok := t.(ScriggoType); ok {
			isScriggoType = true
			break
		}
	}

	if isScriggoType {

		inBase := make([]reflect.Type, len(in))
		outBase := make([]reflect.Type, len(out))

		for i := range in {
			if st, ok := in[i].(ScriggoType); ok {
				inBase[i] = st.Underlying()
			} else {
				inBase[i] = in[i]
			}
		}
		for i := range out {
			if st, ok := out[i].(ScriggoType); ok {
				outBase[i] = st.Underlying()
			} else {
				outBase[i] = out[i]
			}
		}

		return funcType{
			in:   &in,
			out:  &out,
			Type: FuncOf(inBase, outBase, variadic),
		}

	}

	return reflect.FuncOf(in, out, variadic)

}

type funcType struct {
	reflect.Type

	in, out *[]reflect.Type
}

func (x funcType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x funcType) In(i int) reflect.Type {
	if x.in != nil {
		return (*x.in)[i]
	}
	return x.Type.In(i)
}

func (x funcType) Out(i int) reflect.Type {
	if x.out != nil {
		return (*x.out)[i]
	}
	return x.Type.Out(i)
}

func (x funcType) Kind() reflect.Kind {
	return reflect.Func
}

func (x funcType) Name() string {
	return "" // this is a composite type, not a defined type.
}

func (x funcType) Underlying() reflect.Type {
	assertNotScriggoType(x)
	return x.Type
}

func (x funcType) String() string {
	s := "func("
	for i, t := range *x.in {
		s += t.String()
		if i != len(*x.in)-1 {
			s += ", "
		}
	}
	s += ") "
	if len(*x.out) >= 2 {
		s += "("
	}
	for i, t := range *x.out {
		s += t.String()
		if i != len(*x.out)-1 {
			s += ", "
		}
	}
	if len(*x.out) >= 2 {
		s += ")"
	}
	return s
}
