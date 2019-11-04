// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import "reflect"

type scriggoFuncType struct {
	reflect.Type

	in, out *[]reflect.Type
}

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

		return scriggoFuncType{
			in:   &in,
			out:  &out,
			Type: FuncOf(inBase, outBase, variadic),
		}

	}

	return reflect.FuncOf(in, out, variadic)

}

func (st scriggoFuncType) In(i int) reflect.Type {
	if st.in != nil {
		return (*st.in)[i]
	}
	return st.Type.In(i)
}

func (st scriggoFuncType) Out(i int) reflect.Type {
	if st.out != nil {
		return (*st.out)[i]
	}
	return st.Type.Out(i)
}

func (st scriggoFuncType) Kind() reflect.Kind {
	return reflect.Func
}

func (st scriggoFuncType) Name() string {
	return "" // this is a composite type, not a defined type.
}

func (st scriggoFuncType) Underlying() reflect.Type {
	return st.Type
}

func (st scriggoFuncType) String() string {
	s := "func("
	for i, t := range *st.in {
		s += t.String()
		if i != len(*st.in)-1 {
			s += ", "
		}
	}
	s += ") "
	if len(*st.out) >= 2 {
		s += "("
	}
	for i, t := range *st.out {
		s += t.String()
		if i != len(*st.out)-1 {
			s += ", "
		}
	}
	if len(*st.out) >= 2 {
		s += ")"
	}
	return s
}
