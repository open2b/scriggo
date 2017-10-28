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
	`{{a}}`:                                            {tokenStartShow, tokenIdentifier, tokenEndShow},
	`{{ a }}`:                                          {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{\ta\n}}":                                        {tokenStartShow, tokenIdentifier, tokenSemicolon, tokenEndShow},
	"{{\na\t}}":                                        {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{\na;\t}}":                                       {tokenStartShow, tokenIdentifier, tokenSemicolon, tokenEndShow},
	"{% var a = 1 %}":                                  {tokenStartStatement, tokenVar, tokenIdentifier, tokenAssignment, tokenNumber, tokenEndStatement},
	"{% a = 2 %}":                                      {tokenStartStatement, tokenIdentifier, tokenAssignment, tokenNumber, tokenEndStatement},
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
	"{% include \"\" %}":                               {tokenStartStatement, tokenInclude, tokenInterpretedString, tokenEndStatement},
	`a{{b}}c`:                                          {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	`{{a}}c`:                                           {tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	`{{a}}\n`:                                          {tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	`{{a}}{{b}}`:                                       {tokenStartShow, tokenIdentifier, tokenEndShow, tokenStartShow, tokenIdentifier, tokenEndShow},
	"<script></script>":                                {tokenText},
	"<style></style>":                                  {tokenText},
	"<script>{{a}}</script>":                           {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	"<style>{{a}}</style>":                             {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	"<a class=\"{{c}}\"></a>":                          {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	"{{ _ }}":                                          {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ __ }}":                                         {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ _a }}":                                         {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ a5 }}":                                         {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ 3 }}":                                          {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ -3 }}":                                         {tokenStartShow, tokenSubtraction, tokenNumber, tokenEndShow},
	"{{ +3 }}":                                         {tokenStartShow, tokenAddition, tokenNumber, tokenEndShow},
	"{{ 0 }}":                                          {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 2147483647 }}":                                 {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ -2147483648 }}":                                {tokenStartShow, tokenSubtraction, tokenNumber, tokenEndShow},
	"{{ .0 }}":                                         {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 0. }}":                                         {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 0.0 }}":                                        {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 2147483647.2147483647 }}":                      {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ -2147483647.2147483647 }}":                     {tokenStartShow, tokenSubtraction, tokenNumber, tokenEndShow},
	"{{ 2147483647.2147483647214748364721474836472 }}": {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ a + b }}":                                      {tokenStartShow, tokenIdentifier, tokenAddition, tokenIdentifier, tokenEndShow},
	"{{ a - b }}":                                      {tokenStartShow, tokenIdentifier, tokenSubtraction, tokenIdentifier, tokenEndShow},
	"{{ a * b }}":                                      {tokenStartShow, tokenIdentifier, tokenMultiplication, tokenIdentifier, tokenEndShow},
	"{{ a / b }}":                                      {tokenStartShow, tokenIdentifier, tokenDivision, tokenIdentifier, tokenEndShow},
	"{{ a % b }}":                                      {tokenStartShow, tokenIdentifier, tokenModulo, tokenIdentifier, tokenEndShow},
	"{{ ( a ) }}":                                      {tokenStartShow, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndShow},
	"{{ a == b }}":                                     {tokenStartShow, tokenIdentifier, tokenEqual, tokenIdentifier, tokenEndShow},
	"{{ a != b }}":                                     {tokenStartShow, tokenIdentifier, tokenNotEqual, tokenIdentifier, tokenEndShow},
	"{{ a && b }}":                                     {tokenStartShow, tokenIdentifier, tokenAnd, tokenIdentifier, tokenEndShow},
	"{{ a || b }}":                                     {tokenStartShow, tokenIdentifier, tokenOr, tokenIdentifier, tokenEndShow},
	"{{ a < b }}":                                      {tokenStartShow, tokenIdentifier, tokenLess, tokenIdentifier, tokenEndShow},
	"{{ a <= b }}":                                     {tokenStartShow, tokenIdentifier, tokenLessOrEqual, tokenIdentifier, tokenEndShow},
	"{{ a > b }}":                                      {tokenStartShow, tokenIdentifier, tokenGreater, tokenIdentifier, tokenEndShow},
	"{{ a >= b }}":                                     {tokenStartShow, tokenIdentifier, tokenGreaterOrEqual, tokenIdentifier, tokenEndShow},
	"{{ !a }}":                                         {tokenStartShow, tokenNot, tokenIdentifier, tokenEndShow},
	"{{ a[5] }}":                                       {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenRightBrackets, tokenEndShow},
	"{{ a[:] }}":                                       {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenRightBrackets, tokenEndShow},
	"{{ a[:8] }}":                                      {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenNumber, tokenRightBrackets, tokenEndShow},
	"{{ a[3:] }}":                                      {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenColon, tokenRightBrackets, tokenEndShow},
	"{{ a[3:8] }}":                                     {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenColon, tokenNumber, tokenRightBrackets, tokenEndShow},
	"{{ a() }}":                                        {tokenStartShow, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenEndShow},
	"{{ a(1) }}":                                       {tokenStartShow, tokenIdentifier, tokenLeftParenthesis, tokenNumber, tokenRightParenthesis, tokenEndShow},
	"{{ a(1,2) }}":                                     {tokenStartShow, tokenIdentifier, tokenLeftParenthesis, tokenNumber, tokenComma, tokenNumber, tokenRightParenthesis, tokenEndShow},
	"{{ a.b }}":                                        {tokenStartShow, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndShow},
	"{{ \"\" }}":                                       {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\u09AF\" }}":                                {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\u09af\" }}":                                {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\U00008a9e\" }}":                            {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\U0010FFFF\" }}":                            {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\t\" }}":                                    {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\u3C2E\\\"\" }}":                            {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ `` }}":                                         {tokenStartShow, tokenRawString, tokenEndShow},
	"{{ `\\t` }}":                                      {tokenStartShow, tokenRawString, tokenEndShow},
	"{{ ( 1 + 2 ) * 3 }}": {tokenStartShow, tokenLeftParenthesis, tokenNumber, tokenAddition, tokenNumber, tokenRightParenthesis,
		tokenMultiplication, tokenNumber, tokenEndShow},
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
