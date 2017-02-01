//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"testing"
)

var typeTests = map[string][]tokenType{
	``:                             {},
	`a`:                            {tokenText},
	`{`:                            {tokenText},
	`}`:                            {tokenText},
	`{{a}}`:                        {tokenStartShow, tokenIdentifier, tokenEndShow},
	`{{ a }}`:                      {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{\ta\n}}":                    {tokenStartShow, tokenIdentifier, tokenSemicolon, tokenEndShow},
	"{{\na\t}}":                    {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{\na;\t}}":                   {tokenStartShow, tokenIdentifier, tokenSemicolon, tokenEndShow},
	"{% var a = 1 %}":              {tokenStartStatement, tokenVar, tokenIdentifier, tokenAssignment, tokenNumber, tokenEndStatement},
	"{% a = 2 %}":                  {tokenStartStatement, tokenIdentifier, tokenAssignment, tokenNumber, tokenEndStatement},
	"{%for()%}":                    {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenEndStatement},
	"{%\tfor()\n%}":                {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon, tokenEndStatement},
	"{%\tfor a%}":                  {tokenStartStatement, tokenFor, tokenIdentifier, tokenEndStatement},
	"{% for a;\n\t%}":              {tokenStartStatement, tokenFor, tokenIdentifier, tokenSemicolon, tokenEndStatement},
	"{%end%}":                      {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{%\tend\n%}":                  {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% end %}":                    {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% if a %}":                   {tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement},
	"{% extend \"\" %}":            {tokenStartStatement, tokenExtend, tokenInterpretedString, tokenEndStatement},
	"{% region \"\" %}":            {tokenStartStatement, tokenRegion, tokenInterpretedString, tokenEndStatement},
	"{% include \"\" %}":           {tokenStartStatement, tokenInclude, tokenInterpretedString, tokenEndStatement},
	"{% snippet \"\" %}":           {tokenStartStatement, tokenSnippet, tokenInterpretedString, tokenEndStatement},
	`a{{b}}c`:                      {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	`{{a}}c`:                       {tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	`{{a}}\n`:                      {tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	`{{a}}{{b}}`:                   {tokenStartShow, tokenIdentifier, tokenEndShow, tokenStartShow, tokenIdentifier, tokenEndShow},
	"<script></script>":            {tokenText},
	"<style></style>":              {tokenText},
	"<script>{{a}}</script>":       {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	"<style>{{a}}</style>":         {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	"<a class=\"{{c}}\"></a>":      {tokenText, tokenStartShow, tokenIdentifier, tokenEndShow, tokenText},
	"{{ _ }}":                      {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ __ }}":                     {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ _a }}":                     {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ a5 }}":                     {tokenStartShow, tokenIdentifier, tokenEndShow},
	"{{ 3 }}":                      {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ -3 }}":                     {tokenStartShow, tokenSubtraction, tokenNumber, tokenEndShow},
	"{{ +3 }}":                     {tokenStartShow, tokenAddition, tokenNumber, tokenEndShow},
	"{{ 0 }}":                      {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 2147483647 }}":             {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ -2147483648 }}":            {tokenStartShow, tokenSubtraction, tokenNumber, tokenEndShow},
	"{{ .0 }}":                     {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 0. }}":                     {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 0.0 }}":                    {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ 2147483647.2147483647 }}":  {tokenStartShow, tokenNumber, tokenEndShow},
	"{{ -2147483647.2147483647 }}": {tokenStartShow, tokenSubtraction, tokenNumber, tokenEndShow},
	"{{ a + b }}":                  {tokenStartShow, tokenIdentifier, tokenAddition, tokenIdentifier, tokenEndShow},
	"{{ a - b }}":                  {tokenStartShow, tokenIdentifier, tokenSubtraction, tokenIdentifier, tokenEndShow},
	"{{ a * b }}":                  {tokenStartShow, tokenIdentifier, tokenMultiplication, tokenIdentifier, tokenEndShow},
	"{{ a / b }}":                  {tokenStartShow, tokenIdentifier, tokenDivision, tokenIdentifier, tokenEndShow},
	"{{ a % b }}":                  {tokenStartShow, tokenIdentifier, tokenModulo, tokenIdentifier, tokenEndShow},
	"{{ ( a ) }}":                  {tokenStartShow, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndShow},
	"{{ a == b }}":                 {tokenStartShow, tokenIdentifier, tokenEqual, tokenIdentifier, tokenEndShow},
	"{{ a != b }}":                 {tokenStartShow, tokenIdentifier, tokenNotEqual, tokenIdentifier, tokenEndShow},
	"{{ a && b }}":                 {tokenStartShow, tokenIdentifier, tokenAnd, tokenIdentifier, tokenEndShow},
	"{{ a || b }}":                 {tokenStartShow, tokenIdentifier, tokenOr, tokenIdentifier, tokenEndShow},
	"{{ a < b }}":                  {tokenStartShow, tokenIdentifier, tokenLess, tokenIdentifier, tokenEndShow},
	"{{ a <= b }}":                 {tokenStartShow, tokenIdentifier, tokenLessOrEqual, tokenIdentifier, tokenEndShow},
	"{{ a > b }}":                  {tokenStartShow, tokenIdentifier, tokenGreater, tokenIdentifier, tokenEndShow},
	"{{ a >= b }}":                 {tokenStartShow, tokenIdentifier, tokenGreaterOrEqual, tokenIdentifier, tokenEndShow},
	"{{ !a }}":                     {tokenStartShow, tokenNot, tokenIdentifier, tokenEndShow},
	"{{ a[5] }}":                   {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenRightBrackets, tokenEndShow},
	"{{ a[:] }}":                   {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenRightBrackets, tokenEndShow},
	"{{ a[:8] }}":                  {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenNumber, tokenRightBrackets, tokenEndShow},
	"{{ a[3:] }}":                  {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenColon, tokenRightBrackets, tokenEndShow},
	"{{ a[3:8] }}":                 {tokenStartShow, tokenIdentifier, tokenLeftBrackets, tokenNumber, tokenColon, tokenNumber, tokenRightBrackets, tokenEndShow},
	"{{ a() }}":                    {tokenStartShow, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenEndShow},
	"{{ a(1) }}":                   {tokenStartShow, tokenIdentifier, tokenLeftParenthesis, tokenNumber, tokenRightParenthesis, tokenEndShow},
	"{{ a(1,2) }}":                 {tokenStartShow, tokenIdentifier, tokenLeftParenthesis, tokenNumber, tokenComma, tokenNumber, tokenRightParenthesis, tokenEndShow},
	"{{ a.b }}":                    {tokenStartShow, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndShow},
	"{{ \"\" }}":                   {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ \"\\t\" }}":                {tokenStartShow, tokenInterpretedString, tokenEndShow},
	"{{ `` }}":                     {tokenStartShow, tokenRawString, tokenEndShow},
	"{{ `\\t` }}":                  {tokenStartShow, tokenRawString, tokenEndShow},
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
	//"<a class=\"{{c}}\"></a>": {contextHTML, contextAttribute, contextAttribute, contextAttribute, contextHTML},
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
