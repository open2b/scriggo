//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"testing"

	"open2b/template/ast"
)

var typeTests = map[string][]tokenType{
	``:                                                 {},
	`a`:                                                {tokenText},
	`{`:                                                {tokenText},
	`}`:                                                {tokenText},
	`{{a}}`:                                            {tokenStartValue, tokenIdentifier, tokenEndValue},
	`{{ a }}`:                                          {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{\ta\n}}":                                        {tokenStartValue, tokenIdentifier, tokenSemicolon, tokenEndValue},
	"{{\na\t}}":                                        {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{\na;\t}}":                                       {tokenStartValue, tokenIdentifier, tokenSemicolon, tokenEndValue},
	"{% var a = 1 %}":                                  {tokenStartStatement, tokenVar, tokenIdentifier, tokenAssignment, tokenInt, tokenEndStatement},
	"{% a = 2 %}":                                      {tokenStartStatement, tokenIdentifier, tokenAssignment, tokenInt, tokenEndStatement},
	"{%for()%}":                                        {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenEndStatement},
	"{%\tfor()\n%}":                                    {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon, tokenEndStatement},
	"{%\tfor a%}":                                      {tokenStartStatement, tokenFor, tokenIdentifier, tokenEndStatement},
	"{% for a;\n\t%}":                                  {tokenStartStatement, tokenFor, tokenIdentifier, tokenSemicolon, tokenEndStatement},
	"{%end%}":                                          {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{%\tend\n%}":                                      {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% end %}":                                        {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% break %}":                                      {tokenStartStatement, tokenBreak, tokenEndStatement},
	"{% continue %}":                                   {tokenStartStatement, tokenContinue, tokenEndStatement},
	"{% if a %}":                                       {tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement},
	"{% else %}":                                       {tokenStartStatement, tokenElse, tokenEndStatement},
	"{% extend \"\" %}":                                {tokenStartStatement, tokenExtend, tokenInterpretedString, tokenEndStatement},
	"{% region \"\" %}":                                {tokenStartStatement, tokenRegion, tokenInterpretedString, tokenEndStatement},
	"{% show \"\" %}":                                  {tokenStartStatement, tokenShow, tokenInterpretedString, tokenEndStatement},
	"{# comment #}":                                    {tokenComment},
	`a{{b}}c`:                                          {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}c`:                                           {tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}\n`:                                          {tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}{{b}}`:                                       {tokenStartValue, tokenIdentifier, tokenEndValue, tokenStartValue, tokenIdentifier, tokenEndValue},
	"<script></script>":                                {tokenText},
	"<style></style>":                                  {tokenText},
	"<script>{{a}}</script>":                           {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"<style>{{a}}</style>":                             {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"<a class=\"{{c}}\"></a>":                          {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"{{ _ }}":                                          {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ __ }}":                                         {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ _a }}":                                         {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ a5 }}":                                         {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ 3 }}":                                          {tokenStartValue, tokenInt, tokenEndValue},
	"{{ -3 }}":                                         {tokenStartValue, tokenSubtraction, tokenInt, tokenEndValue},
	"{{ +3 }}":                                         {tokenStartValue, tokenAddition, tokenInt, tokenEndValue},
	"{{ 0 }}":                                          {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 2147483647 }}":                                 {tokenStartValue, tokenInt, tokenEndValue},
	"{{ -2147483648 }}":                                {tokenStartValue, tokenSubtraction, tokenInt, tokenEndValue},
	"{{ .0 }}":                                         {tokenStartValue, tokenDecimal, tokenEndValue},
	"{{ 0. }}":                                         {tokenStartValue, tokenDecimal, tokenEndValue},
	"{{ 0.0 }}":                                        {tokenStartValue, tokenDecimal, tokenEndValue},
	"{{ 2147483647.2147483647 }}":                      {tokenStartValue, tokenDecimal, tokenEndValue},
	"{{ -2147483647.2147483647 }}":                     {tokenStartValue, tokenSubtraction, tokenDecimal, tokenEndValue},
	"{{ 2147483647.2147483647214748364721474836472 }}": {tokenStartValue, tokenDecimal, tokenEndValue},
	"{{ a + b }}":                                      {tokenStartValue, tokenIdentifier, tokenAddition, tokenIdentifier, tokenEndValue},
	"{{ a - b }}":                                      {tokenStartValue, tokenIdentifier, tokenSubtraction, tokenIdentifier, tokenEndValue},
	"{{ a * b }}":                                      {tokenStartValue, tokenIdentifier, tokenMultiplication, tokenIdentifier, tokenEndValue},
	"{{ a / b }}":                                      {tokenStartValue, tokenIdentifier, tokenDivision, tokenIdentifier, tokenEndValue},
	"{{ a % b }}":                                      {tokenStartValue, tokenIdentifier, tokenModulo, tokenIdentifier, tokenEndValue},
	"{{ ( a ) }}":                                      {tokenStartValue, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndValue},
	"{{ a == b }}":                                     {tokenStartValue, tokenIdentifier, tokenEqual, tokenIdentifier, tokenEndValue},
	"{{ a != b }}":                                     {tokenStartValue, tokenIdentifier, tokenNotEqual, tokenIdentifier, tokenEndValue},
	"{{ a && b }}":                                     {tokenStartValue, tokenIdentifier, tokenAnd, tokenIdentifier, tokenEndValue},
	"{{ a || b }}":                                     {tokenStartValue, tokenIdentifier, tokenOr, tokenIdentifier, tokenEndValue},
	"{{ a < b }}":                                      {tokenStartValue, tokenIdentifier, tokenLess, tokenIdentifier, tokenEndValue},
	"{{ a <= b }}":                                     {tokenStartValue, tokenIdentifier, tokenLessOrEqual, tokenIdentifier, tokenEndValue},
	"{{ a > b }}":                                      {tokenStartValue, tokenIdentifier, tokenGreater, tokenIdentifier, tokenEndValue},
	"{{ a >= b }}":                                     {tokenStartValue, tokenIdentifier, tokenGreaterOrEqual, tokenIdentifier, tokenEndValue},
	"{{ !a }}":                                         {tokenStartValue, tokenNot, tokenIdentifier, tokenEndValue},
	"{{ a[5] }}":                                       {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInt, tokenRightBrackets, tokenEndValue},
	"{{ a[:] }}":                                       {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenRightBrackets, tokenEndValue},
	"{{ a[:8] }}":                                      {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenInt, tokenRightBrackets, tokenEndValue},
	"{{ a[3:] }}":                                      {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInt, tokenColon, tokenRightBrackets, tokenEndValue},
	"{{ a[3:8] }}":                                     {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInt, tokenColon, tokenInt, tokenRightBrackets, tokenEndValue},
	"{{ a() }}":                                        {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenEndValue},
	"{{ a(1) }}":                                       {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenRightParenthesis, tokenEndValue},
	"{{ a(1,2) }}":                                     {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenComma, tokenInt, tokenRightParenthesis, tokenEndValue},
	"{{ a.b }}":                                        {tokenStartValue, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndValue},
	"{{ \"\" }}":                                       {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u09AF\" }}":                                {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u09af\" }}":                                {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\U00008a9e\" }}":                            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\U0010FFFF\" }}":                            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\t\" }}":                                    {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u3C2E\\\"\" }}":                            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ `` }}":                                         {tokenStartValue, tokenRawString, tokenEndValue},
	"{{ `\\t` }}":                                      {tokenStartValue, tokenRawString, tokenEndValue},
	"{{ ( 1 + 2 ) * 3 }}": {tokenStartValue, tokenLeftParenthesis, tokenInt, tokenAddition, tokenInt, tokenRightParenthesis,
		tokenMultiplication, tokenInt, tokenEndValue},
}

var contextTests = map[string][]context{
	`a`:                           {contextHTML},
	`{{a}}`:                       {contextHTML, contextHTML, contextHTML},
	"<script></script>":           {contextHTML},
	"<style></style>":             {contextHTML},
	"<script>{{a}}</script>{{a}}": {contextHTML, contextScript, contextScript, contextScript, contextHTML, contextHTML, contextHTML, contextHTML},
	"<style>{{a}}</style>{{a}}":   {contextHTML, contextStyle, contextStyle, contextStyle, contextHTML, contextHTML, contextHTML, contextHTML},
}

var positionTests = []struct {
	src string
	pos []ast.Position
}{
	{"a", []ast.Position{
		{1, 1, 0, 0}}},
	{"\n", []ast.Position{
		{1, 1, 0, 0}}},
	{"{{a}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}}},
	{"\n{{a}}", []ast.Position{
		{1, 1, 0, 0},
		{2, 1, 1, 2}, {2, 3, 3, 3}, {2, 4, 4, 5}}},
	{"{{a.b}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 3}, {1, 5, 4, 4}, {1, 6, 5, 6}}},
	{"{{1\t+\n2}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 5, 4, 4}, {2, 1, 6, 6}, {2, 2, 7, 8}}},
	{"{{a}}\n{{b}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}, {1, 6, 5, 5},
		{2, 1, 6, 7}, {2, 3, 8, 8}, {2, 4, 9, 10}}},
	{"{{a}}\n<b\nc=\"{{a}}\">\n{{a}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}, {1, 6, 5, 11},
		{3, 4, 12, 13}, {3, 6, 14, 14}, {3, 7, 15, 16}, {3, 9, 17, 19},
		{4, 1, 20, 21}, {4, 3, 22, 22}, {4, 4, 23, 24}}},
}

func TestLexerTypes(t *testing.T) {
TYPES:
	for source, types := range typeTests {
		var lex = newLexer([]byte(source))
		var i int
		for tok := range lex.tokens {
			if tok.typ == tokenEOF {
				break
			}
			if i >= len(types) {
				t.Errorf("source: %q, unexpected %s\n", source, tok)
				continue TYPES
			}
			if tok.typ != types[i] {
				t.Errorf("source: %q, unexpected %s, expecting %s\n", source, tok, types[i])
				continue TYPES
			}
			i++
		}
		if lex.err != nil {
			t.Errorf("source: %q, error %s\n", source, lex.err)
		}
		if i < len(types) {
			t.Errorf("source: %q, meno types\n", source)
		}
	}
}

func TestLexerContexts(t *testing.T) {
CONTEXTS:
	for source, contexts := range contextTests {
		var lex = newLexer([]byte(source))
		var i int
		for tok := range lex.tokens {
			if tok.typ == tokenEOF {
				break
			}
			if i >= len(contexts) {
				t.Errorf("source: %q, unexpected %s\n", source, tok.ctx)
				continue CONTEXTS
			}
			if tok.ctx != contexts[i] {
				t.Errorf("source: %q, for %s unexpected %s, expecting %s\n", source, tok, tok.ctx, contexts[i])
				continue CONTEXTS
			}
			i++
		}
		if lex.err != nil {
			t.Errorf("source: %q, error %s\n", source, lex.err)
		}
		if i < len(contexts) {
			t.Errorf("source: %q, meno contexts\n", source)
		}
	}
}

func TestPositions(t *testing.T) {
	for _, test := range positionTests {
		var lex = newLexer([]byte(test.src))
		var i int
		for tok := range lex.tokens {
			if tok.typ == tokenEOF {
				break
			}
			pos := test.pos[i]
			if tok.pos.Line != pos.Line {
				t.Errorf("source: %q, token: %s, unexpected line %d, expecting %d\n",
					test.src, tok.String(), tok.pos.Line, pos.Line)
			}
			if tok.pos.Column != pos.Column {
				t.Errorf("source: %q, token: %s, unexpected column %d, expecting %d\n",
					test.src, tok.String(), tok.pos.Column, pos.Column)
			}
			if tok.pos.Start != pos.Start {
				t.Errorf("source: %q, token: %s, unexpected start %d, expecting %d\n",
					test.src, tok.String(), tok.pos.Start, pos.Start)
			}
			if tok.pos.End != pos.End {
				t.Errorf("source: %q, token: %s, unexpected end %d, expecting %d\n",
					test.src, tok.String(), tok.pos.End, pos.End)
			}
			i++
		}
		if lex.err != nil {
			t.Errorf("source: %q, error %s\n", test.src, lex.err)
		}
		if i < len(test.pos) {
			t.Errorf("source: %q, meno lines\n", test.src)
		}
	}
}
