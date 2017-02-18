//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"open2b/decimal"
	"open2b/template/ast"
)

const maxUint = ^uint(0)
const maxInt = int(maxUint >> 1)
const minInt = -maxInt - 1

var errValue = errors.New("error value")

type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("template: %s at %q %s", e.Err, e.Path, e.Pos)
}

type Env struct {
	tree  *ast.Tree
	catch func(e *Error) error
}

func NewEnv(tree *ast.Tree, err func(e *Error) error) *Env {
	if tree == nil {
		panic("template: tree is nil")
	}
	return &Env{tree, err}
}

func (env *Env) Execute(wr io.Writer, vars map[string]interface{}) error {
	if wr == nil {
		return errors.New("template: wr is nil")
	}
	s := state{
		vars:     []map[string]interface{}{builtin, vars, {}},
		catch:    env.catch,
		treepath: env.tree.Path,
	}
	var err error
	extend := getExtendNode(env.tree)
	if extend == nil {
		s.path = env.tree.Path
		err = s.execute(wr, env.tree.Nodes, nil)
	} else {
		if extend.Tree == nil {
			return errors.New("template: extend node is not expanded")
		}
		// legge le region
		regions := map[string]*ast.Region{}
		for _, node := range env.tree.Nodes {
			if r, ok := node.(*ast.Region); ok {
				regions[r.Name] = r
			}
		}
		s.path = extend.Path
		err = s.execute(wr, extend.Tree.Nodes, regions)
	}
	if err != nil {
		if env.catch != nil {
			if err2, ok := err.(*Error); ok {
				err = env.catch(err2)
			}
		}
		return err
	}
	return nil
}

type state struct {
	path     string
	vars     []map[string]interface{}
	catch    func(e *Error) error
	treepath string
}

func (s *state) errorf(node ast.Node, format string, args ...interface{}) error {
	var pos = node.Pos()
	var err = &Error{
		Path: s.path,
		Pos:  ast.Position{pos.Line, pos.Column, pos.Start, pos.End},
		Err:  fmt.Errorf(format, args...),
	}
	if s.catch != nil {
		err2 := s.catch(err)
		if err2 == nil {
			err2 = errValue
		}
		return err2
	}
	return err
}

// execute esegue i nodi nodes.
func (s *state) execute(wr io.Writer, nodes []ast.Node, regions map[string]*ast.Region) error {

	var err error

	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			_, err = io.WriteString(wr, node.Text)
			if err != nil {
				return err
			}

		case *ast.Show:

			expr, err := s.eval(node.Expr)
			if err != nil {
				return err
			}

			var s string
			switch e := expr.(type) {
			case string:
				s = e
			case int:
				s = strconv.Itoa(e)
			case decimal.Dec:
				s = e.String()
				i := len(s) - 1
				for i > 0 && s[i] == '0' {
					i--
				}
				if s[i] == '.' {
					s = s[:i]
				} else {
					s = s[:i+1]
				}
			case bool:
				if e {
					s = "true"
				} else {
					s = "false"
				}
			case []string:
				s = strings.Join(e, ", ")
			case []int:
				buf := make([]string, len(e))
				for i, n := range e {
					buf[i] = strconv.Itoa(n)
				}
				s = strings.Join(buf, ", ")
			case []bool:
				buf := make([]string, len(e))
				for i, b := range e {
					if b {
						buf[i] = "true"
					} else {
						buf[i] = "false"
					}
				}
				s = strings.Join(buf, ", ")
			default:
				if str, ok := e.(fmt.Stringer); ok {
					s = str.String()
				}
			}
			_, err = io.WriteString(wr, s)
			if err != nil {
				return err
			}

		case *ast.If:

			if len(node.Nodes) == 0 {
				continue
			}
			expr, err := s.eval(node.Expr)
			if err != nil {
				return err
			}
			if c, ok := expr.(bool); ok {
				if c {
					err = s.execute(wr, node.Nodes, nil)
					if err != nil {
						return err
					}
				}
			} else if expr != nil {
				panic(fmt.Errorf("non-bool %s (type %T) used as if condition", node.Expr, node.Expr))
			}

		case *ast.For:

			if len(node.Nodes) == 0 {
				continue
			}
			index := ""
			if node.Index != nil {
				index = node.Index.Name
			}
			ident := node.Ident.Name

			expr, err := s.eval(node.Expr)
			if err != nil {
				return err
			}

			av := reflect.ValueOf(expr)
			if !av.IsValid() {
				continue
			}

			var list []interface{}
			if av.Kind() == reflect.Slice {
				if av.Len() == 0 {
					continue
				}
				list = make([]interface{}, av.Len())
				for i := 0; i < len(list); i++ {
					list[i] = av.Index(i).Interface()
				}
			} else {
				list = []interface{}{av.Interface()}
			}

			s.vars = append(s.vars, nil)
			for i, v := range list {
				vars := map[string]interface{}{ident: v}
				if index != "" {
					vars[index] = i
				}
				s.vars[len(s.vars)-1] = vars
				err = s.execute(wr, node.Nodes, nil)
				if err != nil {
					return err
				}
			}
			s.vars = s.vars[:len(s.vars)-1]

		case *ast.Var:

			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if _, ok := vars[name]; ok {
				return fmt.Errorf("variable %q already declared in this block: %#v", name, vars)
			}
			vars[name], err = s.eval(node.Expr)
			if err != nil {
				return err
			}

		case *ast.Assignment:

			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				var vars = s.vars[i]
				if _, ok := vars[name]; ok {
					if i < 2 {
						if i == 0 && name == "len" {
							return fmt.Errorf("use of builtin len not in function call")
						}
						return fmt.Errorf("cannot assign to %s", name)
					}
					vars[name], err = s.eval(node.Expr)
					if err != nil {
						return err
					}
					break
				}
			}
			return fmt.Errorf("variable %s not declared", name)

		case *ast.Region:

			if regions != nil {
				region := regions[node.Name]
				if region != nil {
					path := s.path
					s.path = s.treepath
					err = s.execute(wr, region.Nodes, nil)
					if err != nil {
						return err
					}
					s.path = path
				}
			}

		case *ast.Include:

			if node.Tree == nil {
				return errors.New("include node is not expanded")
			}
			path := s.path
			s.path = node.Path
			err = s.execute(wr, node.Tree.Nodes, nil)
			if err != nil {
				return err
			}
			s.path = path

		}
	}

	return nil
}

// eval valuta una espressione ritornandone il valore.
//
// Se si verifica un errore, chiama s.catch con parametro l'errore.
// Se questa ritorna nil allora ritorna l'errore come valore della
// valutazione, altrimenti ritorna nil e l'errore ritornato da s.catch.
func (s *state) eval(exp ast.Expression) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				if e == errValue {
					value = errValue
				} else {
					err = e
				}
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
	case *ast.Decimal:
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
		switch e.(type) {
		case int:
			return e
		case decimal.Dec:
			return e
		}
		panic(s.errorf(node, "invalid operation: + %T", e))
	case ast.OperatorSubtraction:
		switch n := e.(type) {
		case int:
			return -n
		case decimal.Dec:
			return n.Opposite()
		}
		panic(s.errorf(node, "invalid operation: - %T", e))
	}
	panic("Unknown Unary Operator")
}

// evalBinaryOperator valuta un operatore binario e ne ritorna il valore.
// In caso di errore chiama panic con l'errore come parametro.
func (s *state) evalBinaryOperator(node *ast.BinaryOperator) interface{} {

	var expr1 = s.evalExpression(node.Expr1)
	var expr2 = s.evalExpression(node.Expr2)

	switch node.Op {

	case ast.OperatorEqual:
		if expr1 == nil {
			return expr2 == nil
		}
		if expr2 == nil {
			return expr1 == nil
		}
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
		case decimal.Dec:
			if e2, ok := expr2.(decimal.Dec); ok {
				return e1.ComparedTo(e2) == 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T == %T", expr1, expr2))

	case ast.OperatorNotEqual:
		if expr1 == nil {
			return expr2 != nil
		}
		if expr2 == nil {
			return expr1 != nil
		}
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
		case decimal.Dec:
			if e2, ok := expr2.(decimal.Dec); ok {
				return e1.ComparedTo(e2) != 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T != %T", expr1, expr2))

	case ast.OperatorLess:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T < %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 < e2
			}
		case int:
			switch e2 := expr2.(type) {
			case int:
				return e1 < e2
			case decimal.Dec:
				return e2.ComparedTo(decimal.Int(e1)) > 0
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				return e1.ComparedTo(decimal.Int(e2)) < 0
			case decimal.Dec:
				return e1.ComparedTo(e2) < 0
			}
		}
		panic(fmt.Sprintf("invalid operation: %T < %T", expr1, expr2))

	case ast.OperatorLessOrEqual:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T <= %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 <= e2
			}
		case int:
			switch e2 := expr2.(type) {
			case int:
				return e1 <= e2
			case decimal.Dec:
				return e2.ComparedTo(decimal.Int(e1)) >= 0
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				return e1.ComparedTo(decimal.Int(e2)) <= 0
			case decimal.Dec:
				return e1.ComparedTo(e2) <= 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T <= %T", expr1, expr2))

	case ast.OperatorGreater:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T > %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 > e2
			}
		case int:
			switch e2 := expr2.(type) {
			case int:
				return e1 > e2
			case decimal.Dec:
				return e2.ComparedTo(decimal.Int(e1)) < 0
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				return e1.ComparedTo(decimal.Int(e2)) > 0
			case decimal.Dec:
				return e1.ComparedTo(e2) > 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T > %T", expr1, expr2))

	case ast.OperatorGreaterOrEqual:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T >= %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case string:
			if e2, ok := expr2.(string); ok {
				return e1 >= e2
			}
		case int:
			switch e2 := expr2.(type) {
			case int:
				return e1 >= e2
			case decimal.Dec:
				return e2.ComparedTo(decimal.Int(e1)) <= 0
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				return e1.ComparedTo(decimal.Int(e2)) >= 0
			case decimal.Dec:
				return e1.ComparedTo(e2) >= 0
			}
		}
		panic(s.errorf(node, "invalid operation: %T >= %T", expr1, expr2))

	case ast.OperatorAnd:
		if e1, ok := expr1.(bool); ok {
			if e2, ok := expr2.(bool); ok {
				return e1 && e2
			}
		}
		panic(s.errorf(node, "invalid operation: %T && %T", expr1, expr2))

	case ast.OperatorOr:
		if e1, ok := expr1.(bool); ok {
			if e2, ok := expr2.(bool); ok {
				return e1 || e2
			}
		}
		panic(s.errorf(node, "invalid operation: %T || %T", expr1, expr2))

	case ast.OperatorAddition:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T + %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case string:
			switch e2 := expr2.(type) {
			case string:
				return e1 + e2
			case int:
				return e1 + strconv.Itoa(e2)
			}
		case int:
			switch e2 := expr2.(type) {
			case string:
				return strconv.Itoa(e1) + e2
			case int:
				s := e1 + e2
				if e1 > 0 && e2 > 0 && s < 0 || e1 < 0 && e2 < 0 && s > 0 {
					return decimal.Int(e1).Plus(decimal.Int(e2))
				}
				return s
			case decimal.Dec:
				return e2.Plus(decimal.Int(e1))
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case string:
				return e1.String() + e2
			case int:
				return e1.Plus(decimal.Int(e2))
			case decimal.Dec:
				return e1.Plus(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T + %T", expr1, expr2))

	case ast.OperatorSubtraction:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T - %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			switch e2 := expr2.(type) {
			case int:
				s := e1 - e2
				if (s < e1) != (e2 > 0) {
					return decimal.Int(e1).Minus(decimal.Int(e2))
				}
				return s
			case decimal.Dec:
				return decimal.Int(e1).Minus(e2)
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				return e1.Minus(decimal.Int(e2))
			case decimal.Dec:
				return e1.Minus(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T - %T", expr1, expr2))

	case ast.OperatorMultiplication:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T * %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			switch e2 := expr2.(type) {
			case int:
				var e = e1 * e2
				if e1 != 0 && e/e1 != e2 {
					return decimal.Int(e1).Multiplied(decimal.Int(e2))
				}
				return e
			case decimal.Dec:
				return decimal.Int(e1).Multiplied(e2)
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				if e2 == 0 {
					return 0
				}
				return e1.Multiplied(decimal.Int(e2))
			case decimal.Dec:
				return e1.Multiplied(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T * %T", expr1, expr2))

	case ast.OperatorDivision:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T / %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			switch e2 := expr2.(type) {
			case int:
				if e2 == 0 {
					panic("zero division")
				}
				if e1%e2 == 0 && !(e1 == minInt && e2 == -1) {
					return e1 / e2
				}
				return decimal.Int(e1).Divided(decimal.Int(e2))
			case decimal.Dec:
				if e2.IsZero() {
					panic("zero division")
				}
				return decimal.Int(e1).Divided(e2)
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				if e2 == 0 {
					panic("zero division")
				}
				return e1.Divided(decimal.Int(e2))
			case decimal.Dec:
				if e2.IsZero() {
					panic("zero division")
				}
				return e1.Divided(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T / %T", expr1, expr2))

	case ast.OperatorModulo:
		if expr1 == nil || expr2 == nil {
			panic(s.errorf(node, "invalid operation: %T % %T", expr1, expr2))
		}
		switch e1 := expr1.(type) {
		case int:
			switch e2 := expr2.(type) {
			case int:
				if e2 == 0 {
					panic("zero division")
				}
				return e1 % e2
			case decimal.Dec:
				if e2.IsZero() {
					panic("zero division")
				}
				return decimal.Int(e1).Module(e2)
			}
		case decimal.Dec:
			switch e2 := expr2.(type) {
			case int:
				if e2 == 0 {
					panic("zero division")
				}
				return e1.Module(decimal.Int(e2))
			case decimal.Dec:
				if e2.IsZero() {
					panic("zero division")
				}
				return e1.Module(e2)
			}
		}
		panic(s.errorf(node, "invalid operation: %T % %T", expr1, expr2))

	}

	panic("unknown binary operator")
}

func (s *state) evalSelector(node *ast.Selector) interface{} {
	var e = reflect.ValueOf(s.evalExpression(node.Expr))
	if e.Kind() == reflect.Map {
		return e.MapIndex(reflect.ValueOf(node.Ident)).Interface()
	}
	return nil
}

func (s *state) evalIndex(node *ast.Index) interface{} {
	index, ok := s.evalExpression(node.Index).(int)
	if !ok {
		panic(s.errorf(node, "non-integer slice index %s", node.Index))
	}
	var e = reflect.ValueOf(s.evalExpression(node.Expr))
	if e.Kind() == reflect.Slice {
		if index < 0 || e.Len() <= index {
			return nil
		}
		return e.Index(index).Interface()
	}
	if e.Kind() == reflect.String {
		var s = e.Interface().(string)
		var p = 0
		for _, c := range s {
			if p == index {
				return string(c)
			}
		}
		return nil
	}
	return nil
}

func (s *state) evalSlice(node *ast.Slice) interface{} {
	var l, h int
	if node.Low != nil {
		l = toInt(s.evalExpression(node.Low))
		if l < 0 {
			l = 0
		}
	}
	if node.High != nil {
		h = toInt(s.evalExpression(node.High))
		if h < 0 {
			h = 0
		}
	}
	var e = reflect.ValueOf(s.evalExpression(node.Expr))
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
		if v, ok := s.vars[i][node.Name]; ok {
			if v == errValue {
				panic(v)
			}
			return v
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
				a, ok = arg.(int)
				if !ok {
					if arg == nil {
						panic(s.errorf(node, "cannot use nil as type int in argument to function %s", node.Func))
					} else {
						panic(s.errorf(node, "cannot use %#v (type %T) as type int in argument to function %s", arg, arg, node.Func))
					}
				}
			}
			args[i] = reflect.ValueOf(a)
		case reflect.String:
			var a = ""
			if i < len(node.Args) {
				arg := s.evalExpression(node.Args[i])
				a, ok = arg.(string)
				if !ok {
					if arg == nil {
						panic(s.errorf(node, "cannot use nil as type string in argument to function %s", node.Func))
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
		default:
			panic("unsupported call argument type")
		}
	}

	var vals = fun.Call(args)

	return vals[0].Interface()
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case decimal.Dec:
		i, err := strconv.Atoi(n.Rounded(0, "Down").String())
		if err != nil {
			return 0
		}
		return i
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return 0
		}
		return i
	}
	return 0
}

// getExtendNode ritorna il nodo Extend di un albero.
// Se il nodo non è presente ritorna nil.
func getExtendNode(tree *ast.Tree) *ast.Extend {
	if len(tree.Nodes) == 0 {
		return nil
	}
	if node, ok := tree.Nodes[0].(*ast.Extend); ok {
		return node
	}
	if len(tree.Nodes) > 1 {
		if node, ok := tree.Nodes[1].(*ast.Extend); ok {
			return node
		}
	}
	return nil
}
