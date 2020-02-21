// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compiler implements parsing, type checking and emitting of sources.
//
// A program can be compiled using
//
//	CompileProgram
//
// while a template is compiled through
//
//  CompileTemplate
//
package compiler

import (
	"fmt"

	"scriggo/compiler/ast"
	"scriggo/runtime"
)

// A LimitError is an error returned by the compiler reporting that the
// compilation has reached a limit imposed by the implementation.
type LimitError struct {
	// pos is the position of the function whose body caused the error.
	pos *ast.Position
	// path of the file that caused the error.
	path string
	// msg is the error message. Does not include file/position.
	msg string
}

// newLimitError returns a new LimitError that occurred in the file path inside
// a function declared at the given pos.
func newLimitError(pos *runtime.Position, path, format string, a ...interface{}) *LimitError {
	astPos := &ast.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Start:  pos.Start,
		End:    pos.End,
	}
	return &LimitError{
		pos:  astPos,
		path: path,
		msg:  fmt.Sprintf(format, a...),
	}
}

// Position returns the position of the function whose body caused the
// LimitError.
func (e *LimitError) Position() ast.Position {
	return *e.pos
}

// Path returns the path of the file that caused the LimitError.
func (e *LimitError) Path() string {
	return e.path
}

// Message returns the error message of the LimitError.
func (e *LimitError) Message() string {
	return e.msg
}

// Error implements the interface error for the LimitError.
func (e *LimitError) Error() string {
	return fmt.Sprintf("%s:%s: limit error: %s", e.path, e.pos, e.msg)
}

// limitError implements the interface Scriggo.LimitError.
func (e *LimitError) limitError() {}

// REVIEW: ensure that the error messages of all calls to newLimitError are
// consistent.
