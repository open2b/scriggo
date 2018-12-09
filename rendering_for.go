// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"io"
	"reflect"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

// renderFor renders nodes.
func (r *rendering) renderFor(wr io.Writer, node *ast.For, urlstate *urlState) error {

	index := ""
	if node.Index != nil {
		index = node.Index.Name
	}
	ident := ""
	if node.Ident != nil {
		ident = node.Ident.Name
	}

	expr, err := r.eval(node.Expr1)
	if err != nil {
		if r.handleError(err) {
			return nil
		}
		return err
	}

	if len(node.Nodes) == 0 {
		return nil
	}

	if node.Expr2 == nil {
		// Syntax: for ident in expr
		// Syntax: for index, ident in expr

		setScope := func(i int, v interface{}) {
			vars := scope{ident: v}
			if index != "" {
				vars[index] = i
			}
			r.vars[len(r.vars)-1] = vars
		}

		switch vv := expr.(type) {
		case string:
			size := len(vv)
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			i := 0
			for _, v := range vv {
				setScope(i, string(v))
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
				i++
			}
			r.vars = r.vars[:len(r.vars)-1]
		case HTML:
			size := len(vv)
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			i := 0
			for _, v := range vv {
				setScope(i, HTML(string(v)))
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
				i++
			}
			r.vars = r.vars[:len(r.vars)-1]
		case []string:
			size := len(vv)
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			for i := 0; i < size; i++ {
				setScope(i, vv[i])
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
			r.vars = r.vars[:len(r.vars)-1]
		case []HTML:
			size := len(vv)
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			for i := 0; i < size; i++ {
				setScope(i, vv[i])
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
			r.vars = r.vars[:len(r.vars)-1]
		case []int:
			size := len(vv)
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			for i := 0; i < size; i++ {
				setScope(i, vv[i])
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
			r.vars = r.vars[:len(r.vars)-1]
		case []decimal.Decimal:
			size := len(vv)
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			for i := 0; i < size; i++ {
				setScope(i, vv[i])
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
			r.vars = r.vars[:len(r.vars)-1]
		default:
			av := reflect.ValueOf(expr)
			if !av.IsValid() || av.Kind() != reflect.Slice {
				err = r.errorf(node, "cannot range over %s (type %s)", node.Expr1, typeof(expr))
				if r.handleError(err) {
					return nil
				}
				return err
			}
			size := av.Len()
			if size == 0 {
				return nil
			}
			r.vars = append(r.vars, nil)
			for i := 0; i < size; i++ {
				setScope(i, av.Index(i).Interface())
				err = r.render(wr, node.Nodes, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
			r.vars = r.vars[:len(r.vars)-1]
		}

	} else {
		// Syntax: for index in expr..expr

		expr2, err := r.eval(node.Expr2)
		if err != nil {
			if r.handleError(err) {
				return nil
			}
			return err
		}

		var n1 int
		switch n := expr.(type) {
		case int:
			n1 = n
		case decimal.Decimal:
			n1, err = r.decimalToInt(node.Expr1, n)
		default:
			err = r.errorf(node, "non-integer for range %s", node.Expr1)
		}
		if err != nil {
			if r.handleError(err) {
				return nil
			}
			return err
		}

		var n2 int
		switch n := expr2.(type) {
		case int:
			n2 = n
		case decimal.Decimal:
			n2, err = r.decimalToInt(node.Expr2, n)
		default:
			err = r.errorf(node, "non-integer for range %s", node.Expr2)
		}
		if err != nil {
			if r.handleError(err) {
				return nil
			}
			return err
		}

		step := 1
		if n2 < n1 {
			step = -1
		}

		r.vars = append(r.vars, nil)
		for i := n1; ; i += step {
			r.vars[len(r.vars)-1] = scope{index: i}
			err = r.render(wr, node.Nodes, urlstate)
			if err != nil {
				if err == errBreak {
					break
				}
				if err == errContinue {
					continue
				}
				return err
			}
			if i == n2 {
				break
			}
		}
		r.vars = r.vars[:len(r.vars)-1]

	}

	return nil
}
