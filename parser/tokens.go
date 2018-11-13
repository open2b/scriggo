//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"fmt"

	"open2b/template/ast"
)

// token type.
type tokenType int

const (
	tokenText              tokenType = iota
	tokenStartStatement              // {%
	tokenEndStatement                // %}
	tokenStartValue                  // {{
	tokenEndValue                    // }}
	tokenVar                         // var
	tokenAssignment                  // =
	tokenFor                         // for
	tokenIn                          // in
	tokenBreak                       // break
	tokenContinue                    // continue
	tokenIf                          // if
	tokenElse                        // else
	tokenExtend                      // extend
	tokenImport                      // import
	tokenShow                        // show
	tokenRegion                      // region
	tokenEnd                         // end
	tokenComment                     // comment
	tokenInterpretedString           // "..."
	tokenRawString                   // `...`
	tokenIdentifier                  // customerName
	tokenPeriod                      // .
	tokenLeftParenthesis             // (
	tokenRightParenthesis            // )
	tokenLeftBrackets                // [
	tokenRightBrackets               // ]
	tokenColon                       // :
	tokenComma                       // ,
	tokenSemicolon                   // ;
	tokenRange                       // ..
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
	tokenStartStatement:    "{%",
	tokenEndStatement:      "%}",
	tokenStartValue:        "{{",
	tokenVar:               "var",
	tokenAssignment:        "=",
	tokenEndValue:          "}}",
	tokenFor:               "for",
	tokenIn:                "in",
	tokenBreak:             "break",
	tokenContinue:          "continue",
	tokenIf:                "if",
	tokenElse:              "else",
	tokenExtend:            "extend",
	tokenImport:            "import",
	tokenShow:              "show",
	tokenRegion:            "region",
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
	tokenColon:             ":",
	tokenComma:             ",",
	tokenSemicolon:         ";",
	tokenRange:             "..",
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

// information about a token to return
type token struct {
	typ tokenType     // type
	pos *ast.Position // position in the buffer
	txt []byte        // token text
	ctx ast.Context   // context
	lin int           // line of the lexer when the token was emitted
}

// String returns the string that represents the token.
func (tok token) String() string {
	if tok.typ == tokenText {
		return fmt.Sprintf("%q", tok.txt)
	}
	return tok.typ.String()
}
