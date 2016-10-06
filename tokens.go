//
// Copyright (c) 2016 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"fmt"
)

// context indica il contesto in cui si trova un nodo Show.
type context int

const (
	contextHTML      context = iota // codice HTML
	contextAttribute                // valore di un attributo
	contextScript                   // script
	contextStyle                    // stile
)

func (ctx context) String() string {
	switch ctx {
	case contextHTML:
		return "HTML"
	case contextAttribute:
		return "Attribute"
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
	tokenStartShow                   // {{
	tokenEndShow                     // }}
	tokenVar                         // var
	tokenAssignment                  // =
	tokenFor                         // for
	tokenIf                          // if
	tokenExtend                      // extend
	tokenInclude                     // include
	tokenShow                        // show
	tokenRegion                      // region
	tokenSnippet                     // snippet
	tokenEnd                         // end
	tokenInterpretedString           // "..."
	tokenRawString                   // `...`
	tokenIdentifier                  // customerName
	tokenPeriod                      // .
	tokenLeftParenthesis             // (
	tokenRightParenthesis            // )
	tokenLeftBrackets                // [
	tokenRightBrackets               // ]
	tokenComma                       // ,
	tokenSemicolon                   // ;
	tokenDecimal                     // 12.895
	tokenInt32                       // 81
	tokenInt64                       // 81
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
	tokenStartStatement:    "{%",
	tokenEndStatement:      "%}",
	tokenStartShow:         "{{",
	tokenEndShow:           "}}",
	tokenFor:               "for",
	tokenIf:                "if",
	tokenExtend:            "extend",
	tokenInclude:           "include",
	tokenShow:              "show",
	tokenRegion:            "region",
	tokenSnippet:           "snippet",
	tokenEnd:               "end",
	tokenInterpretedString: "string",
	tokenRawString:         "string",
	tokenIdentifier:        "identifier",
	tokenPeriod:            ".",
	tokenLeftParenthesis:   "(",
	tokenRightParenthesis:  ")",
	tokenLeftBrackets:      "[",
	tokenRightBrackets:     "]",
	tokenComma:             ",",
	tokenSemicolon:         ";",
	tokenDecimal:           "decimal",
	tokenInt32:             "integer",
	tokenInt64:             "integer",
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
	typ tokenType // tipo
	pos int       // posizione nel buffer in byte
	txt []byte    // testo del token
	ctx context   // contesto
}

// String ritorna la stringa che rappresenta il token.
func (tok token) String() string {
	if tok.typ == tokenText {
		return fmt.Sprintf("%q", tok.txt)
	}
	return tok.typ.String()
}
