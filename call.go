// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"errors"
	"fmt"
	"reflect"

	"open2b/template/ast"
	"open2b/template/ast/astutil"
)

// evalCall evaluates a call expression in a single-val context. It returns
// an error if the function
func (r *rendering) evalCall(node *ast.Call) interface{} {
	results, err := r.evalCallN(node, 1)
	if err != nil {
		panic(err)
	}
	return results[0].Interface()
}

// evalCallN evaluates a call expression in n-values context and returns its
// values. It returns an error if n > 0 and the function does not return n
// values.
func (r *rendering) evalCallN(node *ast.Call, n int) ([]reflect.Value, error) {

	// TODO(marco): manage the special case f(g(parameters_of_g)).

	if ident, ok := node.Func.(*ast.Identifier); ok {
		var isBuiltin bool
		for i := len(r.vars) - 1; i >= 0; i-- {
			if r.vars[i] != nil {
				if _, ok := r.vars[i][ident.Name]; ok {
					isBuiltin = i == 0
					break
				}
			}
		}
		if isBuiltin {
			switch ident.Name {
			case "append":
				return r.evalAppend(node, n)
			case "delete":
				return r.evalDelete(node, n)
			case "len":
				return r.evalLen(node, n)
			case "new":
				return r.evalNew(node, n)
			}
		}
	}

	var f, err = r.eval(node.Func)
	if err != nil {
		return nil, err
	}

	// Makes a type conversion.
	if typ, ok := f.(reflect.Type); ok {
		if len(node.Args) == 0 {
			return nil, r.errorf(node, "missing argument to conversion to %s: %s", typ, node)
		}
		if len(node.Args) > 1 {
			return nil, r.errorf(node, "too many arguments to conversion to %s: %s", typ, node)
		}
		v, err := r.convert(node.Args[0], typ)
		if err != nil {
			panic(r.errorf(node.Args[0], "%s", err))
		}
		return []reflect.Value{reflect.ValueOf(v)}, nil
	}

	// Makes a call with reflect.
	fun := reflect.ValueOf(f)
	if !fun.IsValid() {
		if r.isBuiltin("nil", node.Func) {
			return nil, r.errorf(node, "use of untyped nil")
		}
		return nil, r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f))
	}
	if fun.Kind() != reflect.Func {
		return nil, r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f))
	}

	return r.evalReflectCall(node, fun, n)
}

// evalReflectCall evaluates a call expression with reflect in n-values
// context and returns its values. It returns an error if n > 0 and the
// function does not return n values.
func (r *rendering) evalReflectCall(node *ast.Call, fun reflect.Value, n int) ([]reflect.Value, error) {

	if fun.IsNil() {
		return nil, r.errorf(node, "call of nil function")
	}

	typ := fun.Type()

	var numOut = typ.NumOut()
	switch {
	case n == 1 && numOut > 1:
		expr := astutil.CloneExpression(node).(*ast.Call)
		expr.Args = nil
		return nil, r.errorf(node, "multiple-value %s in single-value context", expr)
	case n > 0 && numOut == 0:
		return nil, r.errorf(node, "%s used as value", node)
	case n > 0 && n != numOut:
		return nil, r.errorf(node, "assignment mismatch: %d variables but %d values", n, numOut)
	}

	var numIn = typ.NumIn()
	var isVariadic = typ.IsVariadic()

	if (!isVariadic && len(node.Args) != numIn) || (isVariadic && len(node.Args) < numIn-1) {
		have := "("
		for i, arg := range node.Args {
			if i > 0 {
				have += ", "
			}
			v, err := r.eval(arg)
			if err != nil {
				return nil, err
			}
			have += typeof(v)
		}
		have += ")"
		want := "("
		for i := 0; i < numIn; i++ {
			if i > 0 {
				want += ", "
			}
			if i == numIn-1 && isVariadic {
				want += "..."
			}
			if in := typ.In(i); in.Kind() == reflect.Interface {
				want += "any"
			} else {
				want += typeof(reflect.Zero(in).Interface())
			}
		}
		want += ")"
		if len(node.Args) < numIn {
			return nil, r.errorf(node, "not enough arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want)
		}
		return nil, r.errorf(node, "too many arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want)
	}

	var args = make([]reflect.Value, len(node.Args))

	var lastIn = numIn - 1
	var in reflect.Type

	for i := 0; i < len(node.Args); i++ {

		if i < lastIn || !isVariadic {
			in = typ.In(i)
		} else if i == lastIn {
			in = typ.In(lastIn).Elem()
		}
		kind := in.Kind()

		var arg, err = r.eval(node.Args[i])
		if err != nil {
			return nil, err
		}

		switch a := arg.(type) {
		case nil:
			switch kind {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
				reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
			default:
				return nil, r.errorf(node, "cannot use nil as type %s in argument to %s", in, node.Func)
			}
		case HTML:
			if in != htmlType && in == stringType {
				arg = string(a)
			}
		case ConstantNumber:
			arg, err = a.ToType(in)
			if err != nil {
				if e, ok := err.(errConstantOverflow); ok {
					return nil, r.errorf(node.Args[i], "%s", e)
				}
				return nil, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
			}
		}
		if arg != nil && !reflect.TypeOf(arg).AssignableTo(in) {
			return nil, r.errorf(node.Args[i], "cannot use %s (type %s) as type %s in argument to %s",
				node.Args[i], typeof(arg), in, node.Func)
		}

		if arg == nil {
			args[i] = reflect.Zero(in)
		} else {
			args[i] = reflect.ValueOf(arg)
		}

	}

	values, err := func() (_ []reflect.Value, err error) {
		defer func() {
			if e := recover(); e != nil {
				err = r.errorf(node.Func, "%s", e)
			}
		}()
		return fun.Call(args), nil
	}()

	return values, err
}

func (r *rendering) checkBuiltInParameterCount(node *ast.Call, numIn, numOut, n int) error {
	if len(node.Args) < numIn {
		return r.errorf(node, "missing argument to %s: %s", node.Func, node)
	}
	if len(node.Args) > numIn {
		return r.errorf(node, "too many arguments to %s: %s", node.Func, node)
	}
	if numOut == 0 && n > 0 {
		return r.errorf(node, "%s used as value", node)
	}
	if n != numOut {
		return r.errorf(node, "assignment mismatch: %d variables but %d values", n, numOut)
	}
	return nil
}

var errCannotConvert = errors.New("cannot convert")

// convert converts the val of expr to type typ.
//
// TODO(marco): convert must keep the type aliases, as rune and byte, in error messages.
// TODO(marco): implements conversions for custom numbers.
func (r *rendering) convert(expr ast.Expression, typ reflect.Type) (interface{}, error) {
	value := r.evalExpression(expr)
	converted, err := convert(value, typ)
	if err != nil {
		if err == errCannotConvert {
			if value == nil {
				err = fmt.Errorf("cannot convert nil to type %s", typ)
			} else {
				err = fmt.Errorf("cannot convert %s (type %s) to type %s", expr, typeof(value), typ)
			}
		}
		return nil, err
	}
	return converted, nil
}

// convert converts value to type typ.
func convert(value interface{}, typ reflect.Type) (interface{}, error) {
	if n, ok := value.(ConstantNumber); ok {
		switch typ {
		case stringType:
			switch n.typ {
			case TypeInt, TypeRune:
				n, err := n.ToInt32()
				if err != nil {
					return "\uFFFD", nil
				}
				return string(n), nil
			}
		case intType:
			return n.ToInt()
		case int64Type:
			return n.ToInt64()
		case int32Type:
			return n.ToInt32()
		case int16Type:
			return n.ToInt16()
		case int8Type:
			return n.ToInt8()
		case uintType:
			return n.ToUint()
		case uint64Type:
			return n.ToUint64()
		case uint32Type:
			return n.ToUint32()
		case uint16Type:
			return n.ToUint16()
		case uint8Type:
			return n.ToUint8()
		}
	}
	switch typ {
	case stringType:
		switch v := value.(type) {
		case string, HTML:
			return value, nil
		case int:
			return string(v), nil
		case int64:
			return string(v), nil
		case int32:
			return string(v), nil
		case int16:
			return string(v), nil
		case int8:
			return string(v), nil
		case uint:
			return string(v), nil
		case uint64:
			return string(v), nil
		case uint32:
			return string(v), nil
		case uint16:
			return string(v), nil
		case uint8:
			return string(v), nil
		case []byte:
			return string(v), nil
		case []rune:
			return string(v), nil
		}
	case intType:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int(v.(int64)), nil
	case int64Type:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int32:
			return int64(v), nil
		case int16:
			return int64(v), nil
		case int8:
			return int64(v), nil
		case uint:
			return int64(v), nil
		case uint64:
			return int64(v), nil
		case uint32:
			return int64(v), nil
		case uint16:
			return int64(v), nil
		case uint8:
			return int64(v), nil
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		}
	case int32Type:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int32(v.(int64)), nil
	case int16Type:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int16(v.(int64)), nil
	case int8Type:
		v, err := convert(value, int64Type)
		if err != nil {
			return nil, err
		}
		return int8(v.(int64)), nil
	case uintType:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint(v.(uint64)), nil
	case uint64Type:
		switch v := value.(type) {
		case int64:
			return uint64(v), nil
		case int32:
			return uint64(v), nil
		case int16:
			return uint64(v), nil
		case int8:
			return uint64(v), nil
		case uint:
			return uint64(v), nil
		case uint64:
			return v, nil
		case uint32:
			return uint64(v), nil
		case uint16:
			return uint64(v), nil
		case uint8:
			return uint64(v), nil
		case float64:
			return uint64(v), nil
		case float32:
			return uint64(v), nil
		}
	case uint32Type:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint32(v.(uint64)), nil
	case uint16Type:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint16(v.(uint64)), nil
	case uint8Type:
		v, err := convert(value, uint64Type)
		if err != nil {
			return nil, err
		}
		return uint8(v.(uint64)), nil
	case float64Type:
		switch v := value.(type) {
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int16:
			return float64(v), nil
		case int8:
			return float64(v), nil
		case uint:
			return float64(v), nil
		case uint64:
			return float64(v), nil
		case uint32:
			return float64(v), nil
		case uint16:
			return float64(v), nil
		case uint8:
			return float64(v), nil
		case float32:
			return float64(v), nil
		case ConstantNumber:
			return v.ToFloat64(), nil
		}
	case float32Type:
		v, err := convert(value, float64Type)
		if err != nil {
			return nil, err
		}
		return float32(v.(float64)), nil
	case bytesType:
		switch v := value.(type) {
		case nil:
			return []byte(nil), nil
		case string:
			return []byte(v), nil
		case HTML:
			return []byte(v), nil
		case []byte:
			return v, nil
		}
	case runesType:
		switch v := value.(type) {
		case nil:
			return []rune(nil), nil
		case string:
			return []rune(v), nil
		case HTML:
			return []rune(v), nil
		case []rune:
			return v, nil
		}
	case mapType:
		switch v := value.(type) {
		case nil:
			return map[interface{}]interface{}(nil), nil
		case map[interface{}]interface{}:
			return v, nil
		}
	case sliceType:
		switch v := value.(type) {
		case nil:
			return []interface{}(nil), nil
		case []interface{}:
			return v, nil
		}
	default:
		rv := reflect.ValueOf(value)
		if rv.Type().ConvertibleTo(typ) {
			return rv.Convert(typ).Interface(), nil
		}
	}
	return nil, errCannotConvert
}

// evalAppend evaluates the append builtin function.
func (r *rendering) evalAppend(node *ast.Call, n int) ([]reflect.Value, error) {

	if len(node.Args) == 0 {
		return nil, r.errorf(node, "missing arguments to append")
	}

	slice, err := r.eval(node.Args[0])
	if err != nil {
		return nil, err
	}
	if slice == nil && r.isBuiltin("nil", node.Args[0]) {
		return nil, r.errorf(node, "first argument to append must be typed slice; have untyped nil")
	}

	t := reflect.TypeOf(slice)
	if t.Kind() != reflect.Slice {
		return nil, r.errorf(node, "first argument to append must be slice; have %s", t)
	}
	if n == 0 {
		return nil, r.errorf(node, "%s evaluated but not used", node)
	}
	if n > 1 {
		return nil, r.errorf(node, "assignment mismatch: %d variables but 1 values", n)
	}

	m := len(node.Args) - 1
	if m == 0 {
		return []reflect.Value{reflect.ValueOf(slice)}, nil
	}

	typ := t.Elem()

	if s, ok := slice.([]interface{}); ok {
		var s2 []interface{}
		l, c := len(s), cap(s)
		p := 0
		if l+m <= c {
			s2 = make([]interface{}, m)
			s = s[:c:c]
		} else {
			s2 = make([]interface{}, l+m)
			copy(s2, s)
			s = s2
			p = l
		}
		for i := 1; i < len(node.Args); i++ {
			v, err := r.eval(node.Args[i])
			if err != nil {
				return nil, err
			}
			if n, ok := v.(ConstantNumber); ok {
				v, err = n.ToType(typ)
				if err != nil {
					if e, ok := err.(errConstantOverflow); ok {
						return nil, r.errorf(node.Args[i], "%s", e)
					}
					return nil, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
				}
			}
			s2[p+i-1] = v
		}
		if l+m <= c {
			copy(s[l:], s2)
		}
		return []reflect.Value{reflect.ValueOf(s)}, nil
	}

	sv := reflect.ValueOf(slice)
	var sv2 reflect.Value
	l, c := sv.Len(), sv.Cap()
	p := 0
	if l+n <= c {
		sv2 = reflect.MakeSlice(t, m, m)
		sv = sv.Slice3(0, c, c)
	} else {
		sv2 = reflect.MakeSlice(t, l+m, l+m)
		reflect.Copy(sv2, sv)
		sv = sv2
		p = l
	}
	for i := 1; i < len(node.Args); i++ {
		v, err := r.eval(node.Args[i])
		if err != nil {
			return nil, err
		}
		if n, ok := v.(ConstantNumber); ok {
			v, err = n.ToType(typ)
			if err != nil {
				if e, ok := err.(errConstantOverflow); ok {
					return nil, r.errorf(node.Args[i], "%s", e)
				}
				return nil, r.errorf(node.Args[i], "%s in argument to %s", err, node.Func)
			}
		}
		sv2.Index(p + i - 1).Set(reflect.ValueOf(v))
	}
	if l+m <= c {
		reflect.Copy(sv2.Slice(l, l+m+1), sv2)
	}

	return []reflect.Value{sv}, nil
}

// evalDelete evaluates the delete builtin function.
func (r *rendering) evalDelete(node *ast.Call, n int) ([]reflect.Value, error) {
	err := r.checkBuiltInParameterCount(node, 2, 0, n)
	if err != nil {
		return nil, err
	}
	m := r.evalExpression(node.Args[0])
	k, err := r.mapIndex(node.Args[1])
	if err != nil {
		return nil, err
	}
	// TODO: gestire il caso delle nil map
	switch v := m.(type) {
	case Map:
		delete(v, k)
	case map[string]string:
		k, ok := k.(string)
		if !ok {
			return nil, r.errorf(node, "cannot use %v (type %s) as type string in delete", k, typeof(k))
		}
		delete(v, k)
	case map[string]int:
		k, ok := k.(string)
		if !ok {
			return nil, r.errorf(node, "cannot use %v (type %s) as type string in delete", k, typeof(k))
		}
		delete(v, k)
	case map[string]interface{}:
		k, ok := k.(string)
		if !ok {
			return nil, r.errorf(node, "cannot use %v (type %s) as type string in delete", k, typeof(k))
		}
		delete(v, k)
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Map {
			return nil, r.errorf(node, "first argument to delete must be map; have %s", typeof(m))
		}
		k := reflect.ValueOf(k)
		if k.Type().AssignableTo(rv.Type().Key()) {
			panic(fmt.Sprintf("cannot use %v (type %s) as type %s in delete", k, typeof(k), rv.Type().Key()))
		}
		rv.SetMapIndex(k, reflect.Value{})
	}
	return nil, nil
}

// evalLen evaluates the len builtin function.
func (r *rendering) evalLen(node *ast.Call, n int) ([]reflect.Value, error) {
	err := r.checkBuiltInParameterCount(node, 1, 1, n)
	if err != nil {
		return nil, err
	}
	arg := node.Args[0]
	v := r.evalExpression(arg)
	var length int
	switch s := v.(type) {
	case string:
		length = len(s)
	case HTML:
		length = len(string(s))
	case []interface{}:
		length = len(s)
	case []string:
		length = len(s)
	case []HTML:
		length = len(s)
	case []int:
		length = len(s)
	case []byte:
		length = len(s)
	case []bool:
		length = len(s)
	case Map:
		length = len(s)
	case map[string]string:
		length = len(s)
	case map[string]int:
		length = len(s)
	case map[string]interface{}:
		length = len(s)
	default:
		var rv = reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice:
			length = rv.Len()
		case reflect.Array:
			length = rv.Len()
		case reflect.Map:
			length = rv.Len()
		default:
			return nil, r.errorf(arg, "invalid argument %s (type %s) for len", arg, typeof(v))
		}
	}
	return []reflect.Value{reflect.ValueOf(length)}, nil
}

// evalNew evaluates the new builtin function.
func (r *rendering) evalNew(node *ast.Call, n int) ([]reflect.Value, error) {
	err := r.checkBuiltInParameterCount(node, 1, 1, n)
	if err != nil {
		return nil, err
	}
	typ, err := r.evalType(node.Args[0])
	if err != nil {
		return nil, err
	}
	return []reflect.Value{reflect.New(typ)}, nil
}
