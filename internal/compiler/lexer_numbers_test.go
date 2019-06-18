// +build go1.13

// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"

	"scriggo/internal/compiler/ast"
)

var numbersTests = map[string][]tokenTyp{
	"{{ 0b01 45 }}":       {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0B01 }}":          {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0o51701 }}":       {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0O51701 }}":       {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0x1b6F.c2Ap15 }}": {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 1_2 }}":           {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 1_23_456_789 }}":  {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 1_2.3_4 }}":       {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 1_2.3_4e5_6 }}":   {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 0_0 }}":           {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0_123_456 }}":     {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0x123_456 }}":     {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0b10_01_1 }}":     {tokenStartValue, tokenInt, tokenEndValue},
}

func TestLexerNumbers(t *testing.T) {
	testLexerTypes(t, numbersTests, ast.ContextText)
}
