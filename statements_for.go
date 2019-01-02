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
func (r *rendering) renderFor(wr io.Writer, node ast.Node, urlstate *urlState) error {

	switch n := node.(type) {

	case *ast.For:

		if n.Post != nil && n.Post.Type == ast.AssignmentDeclaration {
			err := r.errorf(n.Post, "cannot declare in post statement of for loop")
			if r.handleError(err) {
				return nil
			}
			return err
		}

		r.vars = append(r.vars, scope{})

		if n.Init != nil {
			err := r.renderAssignment(n.Init)
			if err != nil {
				if r.handleError(err) {
					return nil
				}
				return err
			}
		}

		for {

			if n.Condition != nil {
				cond, err := r.eval(n.Condition)
				if err != nil {
					if r.handleError(err) {
						return nil
					}
					return err
				}
				if !cond.(bool) {
					return nil
				}
			}

			err := r.render(wr, n.Body, urlstate)
			if err != nil {
				if err == errBreak {
					break
				}
				if err != errContinue {
					return err
				}
			}

			if n.Post != nil {
				err = r.renderAssignment(n.Post)
				if err != nil {
					if r.handleError(err) {
						return nil
					}
					return err
				}
			}

		}

		r.vars = r.vars[:len(r.vars)-1]

	case *ast.ForRange:

		var err error

		// addresses contains the addresses of the variables to be assigned.
		var addresses []address
		if len(n.Assignment.Variables) > 0 {
			r.vars = append(r.vars, scope{})
			addresses, err = r.addresses(n.Assignment)
			if err != nil {
				return err
			}
		}

		expr, err := r.eval(n.Assignment.Expr)
		if err != nil {
			if r.handleError(err) {
				return nil
			}
			return err
		}

		if len(n.Body) == 0 {
			return nil
		}

		switch vv := expr.(type) {
		case string:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(string(v))
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case HTML:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(HTML(string(v)))
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case []MutableSlice:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case []string:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case []HTML:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case []decimal.Decimal:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case []int:
			for i, v := range vv {
				if addresses != nil {
					addresses[0].assign(i)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case MutableMap:
			for k, v := range vv {
				if addresses != nil {
					addresses[0].assign(k)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case map[string]interface{}:
			for k, v := range vv {
				if addresses != nil {
					addresses[0].assign(k)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		case map[string]string:
			for k, v := range vv {
				if addresses != nil {
					addresses[0].assign(k)
					if len(addresses) > 1 {
						addresses[1].assign(v)
					}
				}
				err = r.render(wr, n.Body, urlstate)
				if err != nil {
					if err == errBreak {
						break
					}
					if err != errContinue {
						return err
					}
				}
			}
		default:
			av := reflect.ValueOf(expr)
			switch av.Kind() {
			case reflect.Slice:
				length := av.Len()
				for i := 0; i < length; i++ {
					if addresses != nil {
						addresses[0].assign(i)
						if len(addresses) > 1 {
							addresses[1].assign(av.Index(i).Interface())
						}
					}
					err = r.render(wr, n.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						if err != errContinue {
							return err
						}
					}
				}
			case reflect.Map:
				if av.Len() == 0 {
					return nil
				}
				for _, k := range av.MapKeys() {
					if addresses != nil {
						addresses[0].assign(k.Interface())
						if len(addresses) > 1 {
							v := av.MapIndex(k)
							if !v.IsValid() {
								continue
							}
							addresses[1].assign(v.Interface())
						}
					}
					err = r.render(wr, n.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						if err != errContinue {
							return err
						}
					}
				}
			case reflect.Struct, reflect.Ptr:
				keys := structKeys(av)
				if keys == nil {
					err = r.errorf(node, "cannot range over %s (type %s)", n.Assignment.Expr, typeof(expr))
					if r.handleError(err) {
						return nil
					}
					return err
				}
				for k, v := range keys {
					if addresses != nil {
						addresses[0].assign(k)
						if len(addresses) > 1 {
							addresses[1].assign(v.value(av))
						}
					}
					err = r.render(wr, n.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						if err != errContinue {
							return err
						}
					}
				}
			default:
				err = r.errorf(node, "cannot range over %s (type %s)", n.Assignment.Expr, typeof(expr))
				if r.handleError(err) {
					return nil
				}
				return err
			}
		}

		r.vars = r.vars[:len(r.vars)-1]
	}

	return nil
}
