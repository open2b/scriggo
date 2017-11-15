//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"fmt"

	"open2b/template/ast"
)

// context indica il contesto in cui si trova un nodo Show.
type context int

const (
	contextHTML   context = iota // codice HTML
	contextScript                // script
	contextStyle                 // stile
)

func (ctx context) String() string {
	switch ctx {
	case contextHTML:
		return "HTML"
	case contextScript:
		return "Script"
	case contextStyle:
		return "Style"
	}
	panic("invalid context")
}

// tipo di token.
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

// informazioni su un token da restituire
type token struct {
	typ tokenType     // tipo
	pos *ast.Position // posizione nel buffer
	txt []byte        // testo del token
	ctx context       // contesto
	lin int           // linea del lexer quando è stato emesso il token
}

// String ritorna la stringa che rappresenta il token.
func (tok token) String() string {
	if tok.typ == tokenText {
		return fmt.Sprintf("%q", tok.txt)
	}
	return tok.typ.String()
}
