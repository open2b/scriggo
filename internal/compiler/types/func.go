// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

// FuncOf behaves like reflect.FuncOf except when at least once of the
// parameters is a Scriggo type; in such case a new Scriggo function type is
// created and returned as reflect.Type.
func (types *Types) FuncOf(in, out []reflect.Type, variadic bool) reflect.Type {

	// If at least one parameter of the function that is going to be created is
	// a Scriggo type then such function must be represented with a Scriggo
	// type. If not, the resulting function can be represented by the reflect
	// implementation of reflect.Type that can be created with reflect.FuncOf.
	if containsScriggoTypes(in) || containsScriggoTypes(out) {

		inGo := make([]reflect.Type, len(in))
		outGo := make([]reflect.Type, len(out))

		for i := range in {
			if st, ok := in[i].(ScriggoType); ok {
				inGo[i] = st.Underlying()
			} else {
				inGo[i] = in[i]
			}
		}
		for i := range out {
			if st, ok := out[i].(ScriggoType); ok {
				outGo[i] = st.Underlying()
			} else {
				outGo[i] = out[i]
			}
		}

		return funcType{
			in:   &in,
			out:  &out,
			Type: types.FuncOf(inGo, outGo, variadic),
		}

	}

	return reflect.FuncOf(in, out, variadic)

}

// containsScriggoTypes reports whether types contains at least one Scriggo type.
func containsScriggoTypes(types []reflect.Type) bool {
	for _, t := range types {
		if _, ok := t.(ScriggoType); ok {
			return true
		}
	}
	return false
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

func (x funcType) Wrap(v interface{}) interface{} { return wrap(x, v) }

func (x funcType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }
