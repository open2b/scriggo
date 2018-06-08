//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"fmt"
	"html"
	"reflect"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

const maxInt = int64(^uint(0) >> 1)

var varNil = (*struct{})(nil)

// HTML viene usato per le stringhe che contengono codice HTML
// affinché show non le sottoponga ad escape.
// Nelle espressioni si comporta come una stringa.
type HTML string

// Stringer è implementato da qualsiasi valore che si comporta come una stringa.
type Stringer interface {
	String() string
}

// Numberer è implementato da qualsiasi valore che si comporta come un numero.
type Numberer interface {
	Number() decimal.Decimal
}

var stringType = reflect.TypeOf("")
var intType = reflect.TypeOf(0)
var float64Type = reflect.TypeOf(0.0)
var boolType = reflect.TypeOf(false)

var zero = decimal.New(0, 0)
var decimalType = reflect.TypeOf(zero)

// eval valuta una espressione ritornandone il valore.
func (s *state) eval(exp ast.Expression) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				panic(r)
			}
		}
	}()
	return s.evalExpression(exp), nil
}

// evalExpression valuta una espressione ritornandone il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalExpression(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.String:
		return e.Text
	case *ast.Int:
		return e.Value
	case *ast.Number:
		return e.Value
	case *ast.Parentesis:
		return s.evalExpression(e.Expr)
	case *ast.UnaryOperator:
		return s.evalUnaryOperator(e)
	case *ast.BinaryOperator:
		return s.evalBinaryOperator(e)
	case *ast.Identifier:
		return s.evalIdentifier(e)
	case *ast.Call:
		return s.evalCall(e)
	case *ast.Index:
		return s.evalIndex(e)
	case *ast.Slice:
		return s.evalSlice(e)
	case *ast.Selector:
		return s.evalSelector(e)
	default:
		panic(s.errorf(expr, "unexprected node type %#v", expr))
	}
}

// evalUnaryOperator valuta un operatore unario e ne ritorna il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalUnaryOperator(node *ast.UnaryOperator) interface{} {
	var e = s.evalExpression(node.Expr)
	switch node.Op {
	case ast.OperatorNot:
		if b, ok := e.(bool); ok {
			return !b
		}
		panic(s.errorf(node, "invalid operation: ! %T", e))
	case ast.OperatorAddition:
		if _, ok := e.(int); ok {
			return e
		}
		if _, ok := e.(decimal.Decimal); ok {
			return e
		}
		panic(s.errorf(node, "invalid operation: + %T", e))
	case ast.OperatorSubtraction:
		if n, ok := e.(int); ok {
			return -n
		}
		if n, ok := e.(decimal.Decimal); ok {
			return n.Neg()
		}
		panic(s.errorf(node, "invalid operation: - %T", e))
	}
	panic("Unknown Unary Operator")
}

// evalBinaryOperator valuta un operatore binario e ne ritorna il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalBinaryOperator(node *ast.BinaryOperator) interface{} {

	expr1 := s.evalExpression(node.Expr1)

	switch node.Op {

	case ast.OperatorEqual:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(s.errorf(node, "invalid operation: %T == %T", expr1, expr2))
				}
			}()
			if expr2 == nil {
				return reflect.ValueOf(expr1).IsNil()
			}
			return reflect.ValueOf(expr2).IsNil()
		} else {
			expr1, expr2 = htmlToStringType(expr1, expr2)
			switch e1 := expr1.(type) {
			case bool:
				if e2, ok := expr2.(bool); ok {
					return e1 == e2
				}
			case string:
				if e2, ok := expr2.(string); ok {
					return e1 == e2
				}
			case int:
				if e2, ok := expr2.(int); ok {
					return e1 == e2
				}
				if e2, ok := expr2.(decimal.Decimal); ok {
					return decimal.New(int64(e1), 0).Cmp(e2) == 0
				}
			case decimal.Decimal:
				if e2, ok := expr2.(decimal.Decimal); ok {
					return e1.Equal(e2)
				}
				if e2, ok := expr2.(int); ok {
					return e1.Cmp(decimal.New(int64(e2), 0)) == 0
				}
			}
		}
		panic(s.errorf(node, "invalid operation: %T == %T", expr1, expr2))

	case ast.OperatorNotEqual:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			defer func() {
				if recover() != nil {
					panic(s.errorf(node, "invalid operation: %T != %T", expr1, expr2))
				}
			}()
			if expr2 == nil {
				return !reflect.ValueOf(expr1).IsNil()
			}
			return !reflect.ValueOf(expr2).IsNil()
		} else {
			expr1, expr2 = htmlToStringType(expr1, expr2)
			switch e1 := expr1.(type) {
			case bool:
				if e2, ok := expr2.(bool); ok {
					return e1 != e2
				}
			case string:
				if e2, ok := expr2.(string); ok {
					return e1 != e2
				}
			case int:
				if e2, ok := expr2.(int); ok {
					return e1 != e2
				}
				if e2, ok := expr2.(decimal.Decimal); ok {
					return decimal.New(int64(e1), 0).Cmp(e2) != 0
				}
			case decimal.Decimal:
				if e2, ok := expr2.(decimal.Decimal); ok {
					return !e1.Equal(e2)
				}
				if e2, ok := expr2.(int); ok {
					return e1.Cmp(decimal.New(int64(e2), 0)) != 0
				}
			}
		}
		panic(s.errorf(node, "invalid operation: %T != %T", expr1, expr2))

	case ast.OperatorLess:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T < %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 < e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 < e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) < 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) < 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) < 0
			}
		}
		panic(fmt.Sprintf("invalid operation: %T < %T", expr1, expr2))

	case ast.OperatorLessOrEqual:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T <= %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 <= e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 <= e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) <= 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) <= 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) <= 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T <= %T", expr1, expr2))

	case ast.OperatorGreater:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T > %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 > e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 > e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) > 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) > 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) > 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T > %T", expr1, expr2))

	case ast.OperatorGreaterOrEqual:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T >= %T", expr1, expr2))
		}
		expr1, expr2 = htmlToStringType(expr1, expr2)
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 >= e2
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 >= e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Cmp(e2) >= 0
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Cmp(e2) >= 0
			}
			if e2, ok := expr2.(int); ok {
				return e1.Cmp(decimal.New(int64(e2), 0)) >= 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T >= %T", expr1, expr2))

	case ast.OperatorAnd:
		if e1, ok := expr1.(bool); ok {
			if !e1 {
				return false
			}
			expr2 := s.evalExpression(node.Expr2)
			if e2, ok := expr2.(bool); ok {
				return e1 && e2
			}
			panic(s.errorf(node, "invalid operation: %T && %T", expr1, expr2))
		}
		panic(s.errorf(node, "invalid operation: %T && ...", expr1))

	case ast.OperatorOr:
		if e1, ok := expr1.(bool); ok {
			if e1 {
				return true
			}
			expr2 := s.evalExpression(node.Expr2)
			if e2, ok := expr2.(bool); ok {
				return e1 || e2
			}
			panic(s.errorf(node, "invalid operation: %T || %T", expr1, expr2))
		}
		panic(s.errorf(node, "invalid operation: %T || ...", expr1))

	case ast.OperatorAddition:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T + %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case string:
			switch e2 := expr2.(type) {
			case string:
				return e1 + e2
			case HTML:
				return HTML(html.EscapeString(e1) + string(e2))
			}
		case HTML:
			switch e2 := expr2.(type) {
			case string:
				return HTML(string(e1) + html.EscapeString(e2))
			case HTML:
				return HTML(string(e1) + string(e2))
			}
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 + e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Add(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Add(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Add(decimal.New(int64(e2), 0))
			}
		}
		panic(s.errorf(node, "invalid operation: %T + %T", expr1, expr2))

	case ast.OperatorSubtraction:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T - %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 - e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Sub(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Sub(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Sub(decimal.New(int64(e2), 0))
			}
		}
		panic(s.errorf(node, "invalid operation: %T - %T", expr1, expr2))

	case ast.OperatorMultiplication:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T * %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 * e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Mul(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Mul(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Mul(decimal.New(int64(e2), 0))
			}
		}
		panic(s.errorf(node, "invalid operation: %T * %T", expr1, expr2))

	case ast.OperatorDivision:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T / %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 / e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).DivRound(e2, 20)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.DivRound(e2, 20)
			}
			if e2, ok := expr2.(int); ok {
				return e1.DivRound(decimal.New(int64(e2), 0), 20)
			}
		}
		panic(s.errorf(node, "invalid operation: %T / %T", expr1, expr2))

	case ast.OperatorModulo:
		expr2 := s.evalExpression(node.Expr2)
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T %% %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			if e2, ok := expr2.(int); ok {
				return e1 % e2
			}
			if e2, ok := expr2.(decimal.Decimal); ok {
				return decimal.New(int64(e1), 0).Mod(e2)
			}
		case decimal.Decimal:
			if e2, ok := expr2.(decimal.Decimal); ok {
				return e1.Mod(e2)
			}
			if e2, ok := expr2.(int); ok {
				return e1.Mod(decimal.New(int64(e2), 0))
			}
		}
		panic(s.errorf(node, "invalid operation: %T %% %T", expr1, expr2))

	}

	panic("unknown binary operator")
}

func (s *state) evalSelector(node *ast.Selector) interface{} {
	v := s.evalExpression(node.Expr)
	// map
	if v2, ok := v.(map[string]interface{}); ok {
		if v3, ok := v2[node.Ident]; ok {
			return asBase(v3)
		}
		panic(s.errorf(node, "field %q does not exist", node.Ident))
	}
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	switch kind {
	case reflect.Map:
		if rv.Type().Key().Kind() == reflect.String {
			v := rv.MapIndex(reflect.ValueOf(node.Ident))
			if !v.IsValid() {
				panic(s.errorf(node, "field %q does not exist", node.Ident))
			}
			return asBase(v.Interface())
		}
		panic(s.errorf(node, "unsupported vars type"))
	case reflect.Struct:
		value, err := getStructField(rv, node.Ident, s.version)
		if err != nil {
			if err == errFieldNotExist {
				err = s.errorf(node, "field %q does not exist", node.Ident)
			}
			panic(err)
		}
		return asBase(value)
	case reflect.Ptr:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Struct {
			if rv.IsNil() {
				return asBase(rv.Interface())
			}
			value, err := getStructField(reflect.Indirect(rv), node.Ident, s.version)
			if err != nil {
				if err == errFieldNotExist {
					err = s.errorf(node, "field %q does not exist", node.Ident)
				}
				panic(err)
			}
			return asBase(value)
		}
	}
	panic(s.errorf(node, "type %T cannot have fields", v))
}

func (s *state) evalIndex(node *ast.Index) interface{} {
	var i int
	switch index := s.evalExpression(node.Index).(type) {
	case int:
		i = index
	case decimal.Decimal:
		p := index.IntPart()
		if p > maxInt || !decimal.New(p, 0).Equal(index) {
			panic(s.errorf(node, "non-integer slice index %s", node.Index))
		}
		if p < 0 {
			panic(s.errorf(node, "invalid slice index %s (index must be non-negative)", node.Index))
		}
		i = int(p)
	default:
		panic(s.errorf(node, "non-integer slice index %s", node.Index))
	}
	var v = s.evalExpression(node.Expr)
	if v == nil {
		panic(s.errorf(node.Expr, "use of untyped nil"))
	} else if v == varNil {
		panic(s.errorf(node, "index out of range"))
	}
	var e = reflect.ValueOf(v)
	if e.Kind() == reflect.Slice {
		if i < 0 || e.Len() <= i {
			return nil
		}
		return e.Index(i).Interface()
	}
	if e.Kind() == reflect.String {
		var s = e.Interface().(string)
		var p = 0
		for _, c := range s {
			if p == i {
				return string(c)
			}
			p++
		}
		return nil
	}
	return nil
}

func (s *state) evalSlice(node *ast.Slice) interface{} {
	var ok bool
	var l, h int
	if node.Low != nil {
		n := s.evalExpression(node.Low)
		l, ok = n.(int)
		if !ok {
			panic(s.errorf(node, "invalid slice index %s (type %T)", n, n))
		}
		if l < 0 {
			l = 0
		}
	}
	if node.High != nil {
		n := s.evalExpression(node.High)
		h, ok = n.(int)
		if !ok {
			panic(s.errorf(node, "invalid slice index %s (type %T)", n, n))
		}
		if h < 0 {
			h = 0
		}
	}
	var v = s.evalExpression(node.Expr)
	if v == nil {
		panic(s.errorf(node.Expr, "use of untyped nil"))
	} else if v == varNil {
		panic(s.errorf(node, "slice bounds out of range"))
	}
	var e = reflect.ValueOf(v)
	switch e.Kind() {
	case reflect.Slice:
		if node.High == nil || h > e.Len() {
			h = e.Len()
		}
		if h <= l {
			return nil
		}
		return e.Slice(l, h).Interface()
	case reflect.String:
		s := e.String()
		if node.High == nil || h > len(s) {
			h = len(s)
		}
		if h <= l {
			return ""
		}
		if l == 0 && h == len(s) {
			return s
		}
		i := 0
		lb, hb := 0, len(s)
		for ib := range s {
			if i == l {
				lb = ib
				if h == len(s) {
					break
				}
			}
			if i == h {
				hb = ib
				break
			}
			i++
		}
		if lb == 0 && hb == len(s) {
			return s
		}
		return s[lb:hb]
	}
	return nil
}

func (s *state) evalIdentifier(node *ast.Identifier) interface{} {
	for i := len(s.vars) - 1; i >= 0; i-- {
		if s.vars[i] != nil {
			if v, ok := s.vars[i][node.Name]; ok {
				if i == 0 {
					return v
				}
				return asBase(v)
			}
		}
	}
	panic(s.errorf(node, "undefined: %s", node.Name))
}

func (s *state) evalCall(node *ast.Call) interface{} {

	var f = s.evalExpression(node.Func)
	var fun = reflect.ValueOf(f)
	if !fun.IsValid() {
		panic(s.errorf(node, "cannot call non-function %s (type %T)", node.Func, f))
	}
	var typ = fun.Type()
	if typ.Kind() != reflect.Func {
		panic(s.errorf(node, "cannot call non-function %s (type %T)", node.Func, f))
	}

	var numIn = typ.NumIn()
	var args = make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		var ok bool
		var in = typ.In(i)
		if in == decimalType {
			var a = zero
			if i < len(node.Args) {
				arg := s.evalExpression(node.Args[i])
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type decimal in argument to function %s", node.Func))
				}
				switch ar := arg.(type) {
				case decimal.Decimal:
					a = ar
				case int:
					a = decimal.New(int64(ar), 0)
				default:
					panic(s.errorf(node, "cannot use %#v (type %T) as type decimal in argument to function %s", arg, arg, node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
			continue
		}
		switch in.Kind() {
		case reflect.Bool:
			var a = false
			if i < len(node.Args) {
				arg := s.evalExpression(node.Args[i])
				a, ok = arg.(bool)
				if !ok {
					if arg == nil {
						panic(s.errorf(node, "cannot use nil as type bool in argument to function %s", node.Func))
					} else {
						panic(s.errorf(node, "cannot use %#v (type %T) as type bool in argument to function %s", arg, arg, node.Func))
					}
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.Int:
			var a = 0
			if i < len(node.Args) {
				arg := s.evalExpression(node.Args[i])
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type int in argument to function %s", node.Func))
				}
				a, ok = arg.(int)
				if !ok {
					panic(s.errorf(node, "cannot use %#v (type %T) as type int in argument to function %s", arg, arg, node.Func))
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.String:
			var a = ""
			if i < len(node.Args) {
				arg := s.evalExpression(node.Args[i])
				if arg == nil {
					panic(s.errorf(node, "cannot use nil as type string in argument to function %s", node.Func))
				}
				a, ok = arg.(string)
				if !ok {
					rv := reflect.ValueOf(arg)
					if rv.Type().ConvertibleTo(stringType) {
						a = rv.String()
					} else {
						panic(s.errorf(node, "cannot use %#v (type %T) as type string in argument to function %s", arg, arg, node.Func))
					}
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.Interface:
			var arg interface{}
			if i < len(node.Args) {
				arg = s.evalExpression(node.Args[i])
			}
			if arg == nil {
				args[i] = reflect.Zero(in)
			} else {
				args[i] = reflect.ValueOf(arg)
			}
		case reflect.Slice:
			var arg interface{}
			if i < len(node.Args) {
				arg = s.evalExpression(node.Args[i])
			}
			if in == reflect.TypeOf(arg) {
				args[i] = reflect.ValueOf(arg)
			} else {
				args[i] = reflect.Zero(in)
			}
		case reflect.Func:
			var arg interface{}
			if i < len(node.Args) {
				arg = s.evalExpression(node.Args[i])
			}
			if arg == nil {
				args[i] = reflect.Zero(in)
			} else {
				args[i] = reflect.ValueOf(arg)
			}
		default:
			panic(s.errorf(node, "unsupported call argument type %s in function %s", in.Kind(), node.Func))
		}
	}

	var vals = fun.Call(args)
	v := vals[0].Interface()

	if d, ok := v.(int); ok {
		v = decimal.New(int64(d), 0)
	}

	return v
}

// htmlToStringType ritorna e1 e e2 con tipo string al posto di HTML.
// Se non hanno tipo HTML vengono ritornate invariate.
func htmlToStringType(e1, e2 interface{}) (interface{}, interface{}) {
	if e, ok := e1.(HTML); ok {
		e1 = string(e)
	}
	if e, ok := e2.(HTML); ok {
		e2 = string(e)
	}
	return e1, e2
}

func asBase(v interface{}) interface{} {
	if v == nil {
		return varNil
	}
	switch vv := v.(type) {
	// number
	case int:
		return vv
	case float64:
		return decimal.NewFromFloat(vv)
	case decimal.Decimal:
		return v
	case Numberer:
		return vv.Number()
	// string
	case string:
		return v
	case HTML:
		return v
	case Stringer:
		return vv.String()
	// bool
	case bool:
		return v
	default:
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		if rt.ConvertibleTo(stringType) {
			return rv.String()
		} else if rt.ConvertibleTo(intType) {
			return rv.Int()
		} else if rt.ConvertibleTo(float64Type) {
			return decimal.NewFromFloat(rv.Float())
		} else if rt.ConvertibleTo(boolType) {
			return rv.Bool()
		}
	}
	return v
}
