// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"open2b/template/ast"
	"open2b/template/ast/astutil"

	"github.com/cockroachdb/apd"
)

// evalCall evaluates a call expression in a single-value context. It returns
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

	if r.isBuiltin("len", node.Func) {
		err := r.checkBuiltInParameterCount(node, 1, 1, n)
		if err != nil {
			return nil, err
		}
		arg := asBase(r.evalExpression(node.Args[0]))
		length := _len(arg)
		if length == -1 {
			return nil, r.errorf(node.Args[0], "invalid argument %s (type %s) for len", node.Args[0], typeof(arg))
		}
		return []reflect.Value{reflect.ValueOf(length)}, nil
	}

	if r.isBuiltin("delete", node.Func) {
		err := r.checkBuiltInParameterCount(node, 2, 0, n)
		if err != nil {
			return nil, err
		}
		arg := asBase(r.evalExpression(node.Args[0]))
		switch m := arg.(type) {
		case Map:
			k := asBase(r.evalExpression(node.Args[1]))
			_delete(m, k)
		case zero:
			_ = r.evalExpression(node.Args[1])
		default:
			typ := typeof(arg)
			if typ == "map" {
				return nil, r.errorf(node, "cannot delete from non-mutable map")
			} else {
				return nil, r.errorf(node, "first argument to delete must be map; have %s", typ)
			}
		}
		return nil, nil
	}

	var f = r.evalExpression(node.Func)

	if typ, ok := f.(valuetype); ok {
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

	if _, ok := f.(zero); ok {
		return nil, r.errorf(node, "call of nil function")
	}

	var fun = reflect.ValueOf(f)
	if !fun.IsValid() {
		if r.isBuiltin("nil", node.Func) {
			return nil, r.errorf(node, "use of untyped nil")
		}
		return nil, r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f))
	}
	if fun.IsNil() {
		return nil, r.errorf(node, "call of nil function")
	}
	var typ = fun.Type()
	if typ.Kind() != reflect.Func {
		return nil, r.errorf(node, "cannot call non-function %s (type %s)", node.Func, typeof(f))
	}

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
			have += typeof(r.evalExpression(arg))
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
		} else {
			return nil, r.errorf(node, "too many arguments in call to %s\n\thave %s\n\twant %s", node.Func, have, want)
		}
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

		inKind := in.Kind()
		var arg interface{}
		if i < len(node.Args) {
			arg = asBase(r.evalExpression(node.Args[i]))
		}
		if arg == nil {
			if i < len(node.Args) {
				if in == decimalType {
					return nil, r.errorf(node, "cannot use nil as type number in argument to %s", node.Func)
				}
				switch inKind {
				case reflect.Bool, reflect.Int, reflect.String:
					wantType := typeof(reflect.Zero(in).Interface())
					return nil, r.errorf(node, "cannot use nil as type %s in argument to %s", wantType, node.Func)
				}
			}
			args[i] = reflect.Zero(in)
		} else {
			if _, ok := arg.(zero); ok {
				args[i] = reflect.Zero(in)
			} else if inKind == reflect.Interface {
				args[i] = reflect.ValueOf(arg)
			} else if d, ok := arg.(*apd.Decimal); ok && in == decimalType {
				args[i] = reflect.ValueOf(d)
			} else if d, ok := arg.(*apd.Decimal); ok && inKind == reflect.Int {
				args[i] = reflect.ValueOf(decimalToInt(d))
			} else if d, ok := arg.(int); ok && in == decimalType {
				args[i] = reflect.ValueOf(apd.New(int64(d), 0))
			} else if html, ok := arg.(HTML); ok && inKind == reflect.String {
				args[i] = reflect.ValueOf(string(html))
			} else if reflect.TypeOf(arg).AssignableTo(in) {
				args[i] = reflect.ValueOf(arg)
			} else {
				switch inKind {
				case reflect.Ptr:
					if in != decimalType {
						return nil, fmt.Errorf("cannot use %s as function parameter type", inKind)
					}
				case reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64,
					reflect.Uint,
					reflect.Uint8,
					reflect.Uint16,
					reflect.Uint32,
					reflect.Uint64,
					reflect.Uintptr,
					reflect.Complex64,
					reflect.Complex128,
					reflect.UnsafePointer,
					reflect.Float32,
					reflect.Float64:
					return nil, fmt.Errorf("cannot use %s as function parameter type", inKind)
				}
				expectedType := typeof(reflect.Zero(in).Interface())
				return nil, r.errorf(node, "cannot use %s (type %s) as type %s in argument to %s", node.Args[i], typeof(arg), expectedType, node.Func)
			}
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

// convert converts the value of expr to type typ.
func (r *rendering) convert(expr ast.Expression, typ valuetype) (interface{}, error) {
	value := asBase(r.evalExpression(expr))
	switch typ {
	case "string":
		switch v := value.(type) {
		case string, HTML:
			return value, nil
		case *apd.Decimal:
			if v.Cmp(decimalMinInt) == -1 || v.Cmp(decimalMaxInt) == 1 {
				return utf8.RuneError, nil
			}
			p, err := v.Int64()
			if err != nil {
				return utf8.RuneError, nil
			}
			return string(int(p)), nil
		case int:
			return string(v), nil
		case Bytes:
			return string(v), nil
		case []byte:
			return string(v), nil
		case zero:
			return "", nil
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				return convertSliceToString(rv), nil
			}
		}
	case "number":
		switch v := value.(type) {
		case *apd.Decimal, int:
			return v, nil
		case zero:
			return 0, nil
		}
	case "int":
		switch v := value.(type) {
		case *apd.Decimal:
			return decimalToInt(v), nil
		case int:
			return v, nil
		case zero:
			return 0, nil
		}
	case "rune":
		switch v := value.(type) {
		case *apd.Decimal:
			e := new(apd.Decimal)
			_, _ = numberConversionContext.RoundToIntegralValue(e, v)
			_, _ = numberConversionContext.Rem(e, e, decimalMod32)
			i64, _ := e.Int64()
			return int(i64), nil
		case int:
			return int(rune(v)), nil
		case zero:
			return 0, nil
		}
	case "byte":
		switch v := value.(type) {
		case *apd.Decimal:
			return int(decimalToByte(v)), nil
		case int:
			return int(byte(v)), nil
		case zero:
			return 0, nil
		}
	case "bool":
		if _, ok := value.(zero); ok {
			return false, nil
		}
	case "map":
		switch v := value.(type) {
		case nil:
		case zero:
			return Map{}, nil
		case Map:
			return v, nil
		default:
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Map, reflect.Struct:
				return v, nil
			case reflect.Ptr:
				if reflect.Indirect(rv).Kind() == reflect.Struct {
					return v, nil
				}
			}
		}
	case "slice":
		switch v := value.(type) {
		case nil:
		case zero:
			return Slice{}, nil
		case string:
			return []rune(v), nil
		case HTML:
			return []rune(string(v)), nil
		case Slice:
			return v, nil
		default:
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Slice {
				return v, nil
			}
		}
	case "bytes":
		switch v := value.(type) {
		case nil:
		case zero:
			return Bytes{}, nil
		case string:
			return Bytes(v), nil
		case HTML:
			return Bytes(v), nil
		case Bytes:
			return v, nil
		default:
			rt := reflect.TypeOf(v)
			if rt.Kind() == reflect.Slice && rt.Elem().Kind() == reflect.Uint8 {
				return v, nil
			}
		}
	}
	if value == nil {
		return nil, fmt.Errorf("cannot convert nil to type %s", typ)
	}
	if _, ok := value.(zero); ok {
		return nil, fmt.Errorf("cannot convert untyped zero to type %s", typ)
	}
	return nil, fmt.Errorf("cannot convert %s (type %s) to type %s", expr, typeof(value), typ)
}

// convertSliceToString converts s of type slice to a value of type string.
func convertSliceToString(s reflect.Value) string {
	l := s.Len()
	if l == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(l)
	for i := 0; i < l; i++ {
		element := asBase(s.Index(i).Interface())
		if element != nil {
			switch e := element.(type) {
			case *apd.Decimal:
				p := &apd.Decimal{}
				apd.BaseContext.WithPrecision(decPrecision).Floor(p, e)
				if p.Cmp(e) == 0 {
					b.WriteString(p.String())
					continue
				}
			case int:
				b.WriteString(string(e))
				continue
			}
		}
		b.WriteRune(utf8.RuneError)
	}
	return b.String()
}
