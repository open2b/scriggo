// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"strconv"

	"github.com/open2b/scriggo/internal/compiler"
	"github.com/open2b/scriggo/internal/runtime"
)

// Position is a position in the source.
type Position struct {
	Line   int // line starting from 1
	Column int // column in characters starting from 1
	Start  int // index of the first byte
	End    int // index of the last byte
}

// String returns the line and column separated by a colon, for example "37:18".
func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

// BuildError represents a build error.
type BuildError struct {
	err compiler.Error
}

func (err *BuildError) Error() string {
	return err.err.Error()
}

func (err *BuildError) Path() string {
	return err.err.Path()
}

func (err *BuildError) Position() Position {
	pos := err.err.Position()
	return Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
}

func (err *BuildError) Message() string {
	return err.err.Message()
}

// PanicError represents an error that occurs when an executed program or
// template calls the panic builtin.
type PanicError struct {
	p *runtime.Panic
}

// Error returns all currently active panics as a string.
//
// To print only the message, use the String method instead.
func (p *PanicError) Error() string {
	return p.p.Error()
}

// Message returns the message.
func (p *PanicError) Message() interface{} {
	return p.p.Message()
}

// Next returns the next panic in the chain.
func (p *PanicError) Next() *PanicError {
	return &PanicError{p.p.Next()}
}

// Recovered reports whether it has been recovered.
func (p *PanicError) Recovered() bool {
	return p.p.Recovered()
}

// String returns the message as a string.
func (p *PanicError) String() string {
	return p.p.String()
}

// Path returns the path of the file that panicked.
func (p *PanicError) Path() string {
	return p.p.Path()
}

// Position returns the position.
func (p *PanicError) Position() Position {
	pos := p.p.Position()
	return Position{Line: pos.Line, Column: pos.Column, Start: pos.Start, End: pos.End}
}
