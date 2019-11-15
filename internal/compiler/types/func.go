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

	if !containsScriggoType(in) && !containsScriggoType(out) {
		return reflect.FuncOf(in, out, variadic)
	}

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
		in:   types.addFuncParameters(in, input),
		out:  types.addFuncParameters(out, output),
		Type: reflect.FuncOf(inGo, outGo, variadic),
	}

}

type parameters = []reflect.Type

const (
	input  = 0
	output = 1
)

// equalParams reports whether params1 and params2 are equal, i.e. have the same
// length and contain the same elements at the same position.
func equalParams(params1, params2 parameters) bool {
	if len(params1) != len(params2) {
		return false
	}
	for i := range params1 {
		if params1[i] != params2[i] {
			return false
		}
	}
	return true
}

// addFuncParameters adds in to the list of input (inOrOut = 0) or output
// (inOurOut = 1) parameters for a function. If a list with the same elements is
// found, then the pointer to such list is returned. This makes comparisons
// betweens two identical function types declared in two parts of the code
// correct. This methods ensures that every pointer to parameter list is equal
// to another pointer if and only if the parameter list is exactly the same.
func (types *Types) addFuncParameters(params parameters, inOrOut int) *parameters {
	for _, storedParams := range types.funcParams[inOrOut] {
		if equalParams(*storedParams, params) {
			return storedParams
		}
	}
	// Not found.
	newParams := &params
	types.funcParams[inOrOut] = append(types.funcParams[inOrOut], newParams)
	return newParams
}

// containsScriggoType reports whether types contains at least one Scriggo type.
func containsScriggoType(types []reflect.Type) bool {
	for _, t := range types {
		if _, ok := t.(ScriggoType); ok {
			return true
		}
	}
	return false
}

type funcType struct {
	reflect.Type

	// At least one of 'in' and 'out' contains at least one Scriggo type,
	// otherwise there's no need to create a funcType. in and out must be
	// pointers because a funcType value must be comparable.
	in, out *[]reflect.Type
}

func (x funcType) AssignableTo(T reflect.Type) bool {
	return x == T
}

func (x funcType) Implements(u reflect.Type) bool {
	if u.Kind() != reflect.Interface {
		panic("expected reflect.Interface")
	}
	return u.NumMethod() == 0
}

func (x funcType) In(i int) reflect.Type {
	if x.in != nil {
		return (*x.in)[i]
	}
	return x.Type.In(i)
}

func (x funcType) Kind() reflect.Kind {
	return reflect.Func
}

func (x funcType) Name() string {
	return "" // composite types do not have a name.
}

func (x funcType) Out(i int) reflect.Type {
	if x.out != nil {
		return (*x.out)[i]
	}
	return x.Type.Out(i)
}

// TODO: does this function handle variadic functions properly?
func (x funcType) String() string {
	s := "func("
	for i, t := range *x.in {
		if i > 0 {
			s += ", "
		}
		s += t.String()
	}
	s += ") "
	if len(*x.out) >= 2 {
		s += "("
	}
	for i, t := range *x.out {
		if i > 0 {
			s += ", "
		}
		s += t.String()
	}
	if len(*x.out) >= 2 {
		s += ")"
	}
	return s
}

// Underlying implement the interface runtime.Wrapper.
func (x funcType) Underlying() reflect.Type {
	assertNotScriggoType(x.Type)
	return x.Type
}

// Unwrap implement the interface runtime.Wrapper.
func (x funcType) Unwrap(v reflect.Value) (reflect.Value, bool) { return unwrap(x, v) }

// Wrap implement the interface runtime.Wrapper.
func (x funcType) Wrap(v reflect.Value) reflect.Value { return wrap(x, v) }
