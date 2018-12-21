// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"

	"open2b/template/ast"
)

// Token type.
type tokenType int

const (
	tokenText              tokenType = iota
	tokenStartURL                    // start url
	tokenEndURL                      // and url
	tokenStartStatement              // {%
	tokenEndStatement                // %}
	tokenStartValue                  // {{
	tokenEndValue                    // }}
	tokenDeclaration                 // :=
	tokenAssignment                  // =
	tokenFor                         // for
	tokenIn                          // in
	tokenBreak                       // break
	tokenContinue                    // continue
	tokenMap                         // map
	tokenSlice                       // slice
	tokenIf                          // if
	tokenElse                        // else
	tokenExtends                     // extends
	tokenImport                      // import
	tokenInclude                     // include
	tokenShow                        // show
	tokenMacro                       // macro
	tokenEnd                         // end
	tokenComment                     // comment
	tokenInterpretedString           // "abc"
	tokenRawString                   // `abc`
	tokenIdentifier                  // customerName
	tokenPeriod                      // .
	tokenLeftParenthesis             // (
	tokenRightParenthesis            // )
	tokenLeftBrackets                // [
	tokenRightBrackets               // ]
	tokenLeftBraces                  // {
	tokenRightBraces                 // }
	tokenColon                       // :
	tokenComma                       // ,
	tokenSemicolon                   // ;
	tokenRange                       // ..
	tokenEllipses                    // ...
	tokenNumber                      // 12.895
	tokenEqual                       // ==
	tokenNotEqual                    // !=
	tokenNot                         // !
	tokenLess                        // <
	tokenLessOrEqual                 // <=
	tokenGreater                     // >
	tokenGreaterOrEqual              // >=
	tokenAnd                         // &&
	tokenOr                          // ||
	tokenAddition                    // +
	tokenSubtraction                 // -
	tokenMultiplication              // *
	tokenDivision                    // /
	tokenModulo                      // %
	tokenEOF                         // eof
)

var tokenString = map[tokenType]string{
	tokenText:              "text",
	tokenStartURL:          "start url",
	tokenEndURL:            "end url",
	tokenStartStatement:    "{%",
	tokenEndStatement:      "%}",
	tokenStartValue:        "{{",
	tokenDeclaration:       ":=",
	tokenAssignment:        "=",
	tokenEndValue:          "}}",
	tokenFor:               "for",
	tokenIn:                "in",
	tokenBreak:             "break",
	tokenContinue:          "continue",
	tokenMap:               "map",
	tokenSlice:             "slice",
	tokenIf:                "if",
	tokenElse:              "else",
	tokenExtends:           "extends",
	tokenImport:            "import",
	tokenInclude:           "include",
	tokenShow:              "show",
	tokenMacro:             "macro",
	tokenEnd:               "end",
	tokenComment:           "comment",
	tokenInterpretedString: "string",
	tokenRawString:         "string",
	tokenIdentifier:        "identifier",
	tokenPeriod:            ".",
	tokenLeftParenthesis:   "(",
	tokenRightParenthesis:  ")",
	tokenLeftBrackets:      "[",
	tokenRightBrackets:     "]",
	tokenLeftBraces:        "{",
	tokenRightBraces:       "}",
	tokenColon:             ":",
	tokenComma:             "comma",
	tokenSemicolon:         "semicolon",
	tokenRange:             "..",
	tokenEllipses:          "...",
	tokenNumber:            "number",
	tokenEqual:             "==",
	tokenNotEqual:          "!=",
	tokenNot:               "!",
	tokenLess:              "<",
	tokenLessOrEqual:       "<=",
	tokenGreater:           ">",
	tokenGreaterOrEqual:    ">=",
	tokenAnd:               "&&",
	tokenOr:                "||",
	tokenAddition:          "+",
	tokenSubtraction:       "-",
	tokenMultiplication:    "*",
	tokenDivision:          "/",
	tokenModulo:            "%",
	tokenEOF:               "EOF",
}

func (tt tokenType) String() string {
	if s, ok := tokenString[tt]; ok {
		return s
	}
	panic("invalid token type")
}

// Information about a token to return.
type token struct {
	typ tokenType     // type
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
