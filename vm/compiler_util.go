// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import "scrigo/ast"

// isNil indicates if expr is the nil identifier.
func isNil(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	if !ok {
		return false
	}
	return ident.Name == "nil"
}

// isBlankIdentifier indicates if expr is an identifier representing the blank
// identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}
