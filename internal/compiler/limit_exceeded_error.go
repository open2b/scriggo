// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// A LimitExceededError is an error returned by the compiler reporting that the
// compilation has exceeded a limit imposed by the implementation.
type LimitExceededError struct {
	// pos is the position of the function that cannot be compiled.
	pos *ast.Position
	// path of file in which the function is defined.
	path string
	// msg is the error message. Does not include file/position.
	msg string
}

// newLimitExceededError returns a new LimitError that occurred in the file path
// inside a function declared at the given pos.
func newLimitExceededError(pos *runtime.Position, path, format string, a ...interface{}) *LimitExceededError {
	astPos := &ast.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Start:  pos.Start,
		End:    pos.End,
	}
	return &LimitExceededError{
		pos:  astPos,
		path: path,
		msg:  fmt.Sprintf(format, a...),
	}
}

// Position returns the position of the function that caused the LimitError.
func (e *LimitExceededError) Position() ast.Position {
	return *e.pos
}

// Path returns the path of the source code that caused the LimitError.
func (e *LimitExceededError) Path() string {
	return e.path
}

// Message returns the error message of the LimitError.
func (e *LimitExceededError) Message() string {
	return e.msg
}

// Error implements the interface error for the LimitError.
func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.path, e.pos, e.msg)
}
