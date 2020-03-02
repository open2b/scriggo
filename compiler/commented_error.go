// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"strings"
	"unicode/utf8"
)

// A value with type commentedError contains an error that will be rendered as a
// comment.
//
// If the language where the commentedError value is rendered does not support
// comments then such value is rendered as the empty string.
type commentedError struct{ Err error }

var commentedErrorType = reflect.TypeOf((*commentedError)(nil)).Elem()

func (ce commentedError) CSS() string {
	if ce.Err == nil {
		return ""
	}
	// See https://drafts.csswg.org/css-syntax-3/#consume-comment.
	msg := ce.Err.Error()
	msg = toValidUTF8(msg)
	msg = strings.ReplaceAll(msg, "*/", "* /")
	return "/* " + msg + " */"

}

func (ce commentedError) HTML() string {
	if ce.Err == nil {
		return ""
	}
	// See https://html.spec.whatwg.org/multipage/syntax.html#comments.
	msg := " " + ce.Err.Error() + " "
	msg = toValidUTF8(msg)
	msg = strings.NewReplacer(
		"<!--", "< !--",
		"-->", "-- >",
		"--!>", "--! >",
	).Replace(msg)
	return "<!--" + msg + "-->"
}

func (ce commentedError) JavaScript() string {
	if ce.Err == nil {
		return ""
	}
	// See https://www.ecma-international.org/ecma-262/5.1/#sec-7.4
	msg := ce.Err.Error()
	msg = toValidUTF8(msg)
	msg = strings.ReplaceAll(msg, "*/", "* /")
	return "/* " + msg + " */"
}

func (ce commentedError) String() string {
	return ""
}

func toValidUTF8(in string) string {
	if utf8.ValidString(in) {
		return in
	}
	out := &strings.Builder{}
	for _, r := range in {
		if r == 0xFFFD {
			out.WriteString("�")
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
