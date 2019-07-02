// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"

	"scriggo/internal/compiler/ast"
)

// Token type.
type tokenTyp int

const (
	tokenText                     tokenTyp = iota
	tokenShebangLine                       // #!
	tokenStartURL                          // start url
	tokenEndURL                            // and url
	tokenStartStatement                    // {%
	tokenEndStatement                      // %}
	tokenStartValue                        // {{
	tokenEndValue                          // }}
	tokenDeclaration                       // :=
	tokenSimpleAssignment                  // =
	tokenAdditionAssignment                // +=
	tokenSubtractionAssignment             // -=
	tokenMultiplicationAssignment          // *=
	tokenDivisionAssignment                // /=
	tokenModuloAssignment                  // %=
	tokenAndAssignment                     // &=
	tokenOrAssignment                      // |=
	tokenXorAssignment                     // ^=
	tokenAndNotAssignment                  // &^=
	tokenLeftShiftAssignment               // <<=
	tokenRightShiftAssignment              // >>=
	tokenPackage                           // package
	tokenFor                               // for
	tokenIn                                // in
	tokenRange                             // range
	tokenBreak                             // break
	tokenContinue                          // continue
	tokenSwitch                            // switch
	tokenCase                              // case
	tokenDefault                           // default
	tokenFallthrough                       // fallthrough
	tokenSelect                            // select
	tokenType                              // type
	tokenInterface                         // interface
	tokenMap                               // map
	tokenChan                              // chan
	tokenIf                                // if
	tokenElse                              // else
	tokenDefer                             // defer
	tokenGo                                // go
	tokenGoto                              // goto
	tokenExtends                           // extends
	tokenImport                            // import
	tokenInclude                           // include
	tokenShow                              // show
	tokenMacro                             // macro
	tokenFunc                              // func
	tokenReturn                            // return
	tokenEnd                               // end
	tokenVar                               // var
	tokenConst                             // const
	tokenComment                           // comment
	tokenInterpretedString                 // "abc"
	tokenRawString                         // `abc`
	tokenRune                              // 'a'
	tokenIdentifier                        // customerName
	tokenPeriod                            // .
	tokenLeftParenthesis                   // (
	tokenRightParenthesis                  // )
	tokenLeftBrackets                      // [
	tokenRightBrackets                     // ]
	tokenLeftBraces                        // {
	tokenRightBraces                       // }
	tokenColon                             // :
	tokenComma                             // ,
	tokenSemicolon                         // ;
	tokenEllipses                          // ...
	tokenFloat                             // 12.895
	tokenInt                               // 18
	tokenImaginary                         // 12.895i
	tokenEqual                             // ==
	tokenNotEqual                          // !=
	tokenNot                               // !
	tokenAnd                               // &
	tokenOr                                // |
	tokenLess                              // <
	tokenLessOrEqual                       // <=
	tokenGreater                           // >
	tokenGreaterOrEqual                    // >=
	tokenAndAnd                            // &&
	tokenOrOr                              // ||
	tokenAddition                          // +
	tokenSubtraction                       // -
	tokenMultiplication                    // *
	tokenDivision                          // /
	tokenModulo                            // %
	tokenIncrement                         // ++
	tokenDecrement                         // --
	tokenArrow                             // <-
	tokenXor                               // ^
	tokenAndNot                            // &^
	tokenLeftShift                         // <<
	tokenRightShift                        // >>
	tokenStruct                            // struct
	tokenEOF                               // eof
)

var tokenString = map[tokenTyp]string{
	tokenText:                     "text",
	tokenShebangLine:              "#!",
	tokenStartURL:                 "start url",
	tokenEndURL:                   "end url",
	tokenStartStatement:           "{%",
	tokenEndStatement:             "%}",
	tokenStartValue:               "{{",
	tokenEndValue:                 "}}",
	tokenDeclaration:              ":=",
	tokenSimpleAssignment:         "=",
	tokenAdditionAssignment:       "+=",
	tokenSubtractionAssignment:    "-=",
	tokenMultiplicationAssignment: "*=",
	tokenDivisionAssignment:       "/=",
	tokenModuloAssignment:         "%=",
	tokenAndAssignment:            "&=",
	tokenOrAssignment:             "|=",
	tokenXorAssignment:            "^=",
	tokenAndNotAssignment:         "&^=",
	tokenLeftShiftAssignment:      "<<=",
	tokenRightShiftAssignment:     ">>=",
	tokenPackage:                  "package",
	tokenFor:                      "for",
	tokenIn:                       "in",
	tokenRange:                    "range",
	tokenBreak:                    "break",
	tokenContinue:                 "continue",
	tokenSwitch:                   "switch",
	tokenCase:                     "case",
	tokenDefault:                  "default",
	tokenFallthrough:              "fallthrough",
	tokenSelect:                   "select",
	tokenType:                     "type",
	tokenInterface:                "interface",
	tokenMap:                      "map",
	tokenChan:                     "chan",
	tokenIf:                       "if",
	tokenElse:                     "else",
	tokenDefer:                    "defer",
	tokenGo:                       "go",
	tokenGoto:                     "goto",
	tokenExtends:                  "extends",
	tokenImport:                   "import",
	tokenInclude:                  "include",
	tokenShow:                     "show",
	tokenFunc:                     "func",
	tokenReturn:                   "return",
	tokenMacro:                    "macro",
	tokenEnd:                      "end",
	tokenVar:                      "var",
	tokenConst:                    "const",
	tokenComment:                  "comment",
	tokenInterpretedString:        "string",
	tokenRawString:                "string",
	tokenRune:                     "rune",
	tokenIdentifier:               "identifier",
	tokenPeriod:                   ".",
	tokenLeftParenthesis:          "(",
	tokenRightParenthesis:         ")",
	tokenLeftBrackets:             "[",
	tokenRightBrackets:            "]",
	tokenLeftBraces:               "{",
	tokenRightBraces:              "}",
	tokenColon:                    ":",
	tokenComma:                    "comma",
	tokenSemicolon:                "semicolon",
	tokenEllipses:                 "...",
	tokenFloat:                    "float",
	tokenInt:                      "int",
	tokenImaginary:                "imaginary",
	tokenEqual:                    "==",
	tokenNotEqual:                 "!=",
	tokenNot:                      "!",
	tokenAnd:                      "&",
	tokenOr:                       "|",
	tokenLess:                     "<",
	tokenLessOrEqual:              "<=",
	tokenGreater:                  ">",
	tokenGreaterOrEqual:           ">=",
	tokenAndAnd:                   "&&",
	tokenOrOr:                     "||",
	tokenAddition:                 "+",
	tokenSubtraction:              "-",
	tokenMultiplication:           "*",
	tokenDivision:                 "/",
	tokenModulo:                   "%",
	tokenIncrement:                "++",
	tokenDecrement:                "--",
	tokenArrow:                    "<-",
	tokenXor:                      "^",
	tokenAndNot:                   "&^",
	tokenLeftShift:                "<<",
	tokenRightShift:               ">>",
	tokenStruct:                   "struct",
	tokenEOF:                      "EOF",
}

func (tt tokenTyp) String() string {
	if s, ok := tokenString[tt]; ok {
		return s
	}
	panic("invalid token type")
}

// Information about a token to return.
type token struct {
	typ tokenTyp      // type
	pos *ast.Position // position in the buffer
	txt []byte        // token text
	ctx ast.Context   // context
	tag string        // tag name
	att string        // attribute
	lin int           // line of the lexer when the token was emitted
}

// String returns the string that represents the token.
func (tok token) String() string {
	if tok.typ == tokenText {
		return fmt.Sprintf("%q", tok.txt)
	}
	return tok.typ.String()
}

// isTokenAssignment indicates whether tok is an assignment token.
func isAssignmentToken(tok token) bool {
	_, ok := assignmentType(tok)
	return ok
}

// assignmentType returns the assignment type of the token tok and true. If
// tok is not an assignment token returns 0 and false.
func assignmentType(tok token) (ast.AssignmentType, bool) {
	switch tok.typ {
	case tokenSimpleAssignment:
		return ast.AssignmentSimple, true
	case tokenDeclaration:
		return ast.AssignmentDeclaration, true
	case tokenAdditionAssignment:
		return ast.AssignmentAddition, true
	case tokenSubtractionAssignment:
		return ast.AssignmentSubtraction, true
	case tokenMultiplicationAssignment:
		return ast.AssignmentMultiplication, true
	case tokenDivisionAssignment:
		return ast.AssignmentDivision, true
	case tokenModuloAssignment:
		return ast.AssignmentModulo, true
	case tokenAndAssignment:
		return ast.AssignmentAnd, true
	case tokenOrAssignment:
		return ast.AssignmentOr, true
	case tokenXorAssignment:
		return ast.AssignmentXor, true
	case tokenAndNotAssignment:
		return ast.AssignmentAndNot, true
	case tokenLeftShiftAssignment:
		return ast.AssignmentLeftShift, true
	case tokenRightShiftAssignment:
		return ast.AssignmentRightShift, true
	case tokenIncrement:
		return ast.AssignmentIncrement, true
	case tokenDecrement:
		return ast.AssignmentDecrement, true
	}
	return 0, false
}
