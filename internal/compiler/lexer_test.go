// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"

	"scriggo/internal/compiler/ast"
)

var typeTests = map[string][]tokenTyp{
	``:                             {},
	`a`:                            {tokenText},
	`{`:                            {tokenText},
	`}`:                            {tokenText},
	`{{a}}`:                        {tokenStartValue, tokenIdentifier, tokenEndValue},
	`{{ a }}`:                      {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{\ta\n}}":                    {tokenStartValue, tokenIdentifier, tokenSemicolon, tokenEndValue},
	"{{\na\t}}":                    {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{\na;\t}}":                   {tokenStartValue, tokenIdentifier, tokenSemicolon, tokenEndValue},
	"{% a := 1 %}":                 {tokenStartStatement, tokenIdentifier, tokenDeclaration, tokenInt, tokenEndStatement},
	"{% a = 2 %}":                  {tokenStartStatement, tokenIdentifier, tokenSimpleAssignment, tokenInt, tokenEndStatement},
	"{% a, ok := b.c %}":           {tokenStartStatement, tokenIdentifier, tokenComma, tokenIdentifier, tokenDeclaration, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndStatement},
	"{% a, ok = b.c %}":            {tokenStartStatement, tokenIdentifier, tokenComma, tokenIdentifier, tokenSimpleAssignment, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndStatement},
	"{%for()%}":                    {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenEndStatement},
	"{%\tfor()\n%}":                {tokenStartStatement, tokenFor, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon, tokenEndStatement},
	"{%\tfor a%}":                  {tokenStartStatement, tokenFor, tokenIdentifier, tokenEndStatement},
	"{% for a;\n\t%}":              {tokenStartStatement, tokenFor, tokenIdentifier, tokenSemicolon, tokenEndStatement},
	"{% for in %}":                 {tokenStartStatement, tokenFor, tokenIn, tokenEndStatement},
	"{% for range %}":              {tokenStartStatement, tokenFor, tokenRange, tokenEndStatement},
	"{%end%}":                      {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{%\tend\n%}":                  {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% end %}":                    {tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% break %}":                  {tokenStartStatement, tokenBreak, tokenEndStatement},
	"{% continue %}":               {tokenStartStatement, tokenContinue, tokenEndStatement},
	"{% if a %}":                   {tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement},
	"{% if a = b; a %}":            {tokenStartStatement, tokenIf, tokenIdentifier, tokenSimpleAssignment, tokenIdentifier, tokenSemicolon, tokenIdentifier, tokenEndStatement},
	"{% if a, ok = b.c; a %}":      {tokenStartStatement, tokenIf, tokenIdentifier, tokenComma, tokenIdentifier, tokenSimpleAssignment, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenSemicolon, tokenIdentifier, tokenEndStatement},
	"{% case 42 %}":                {tokenStartStatement, tokenCase, tokenInt, tokenEndStatement},
	"{% case a %}":                 {tokenStartStatement, tokenCase, tokenIdentifier, tokenEndStatement},
	"{% case a < 20 %}":            {tokenStartStatement, tokenCase, tokenIdentifier, tokenLess, tokenInt, tokenEndStatement},
	"{% case a, b %}":              {tokenStartStatement, tokenCase, tokenIdentifier, tokenComma, tokenIdentifier, tokenEndStatement},
	"{% case int, rune %}":         {tokenStartStatement, tokenCase, tokenIdentifier, tokenComma, tokenIdentifier, tokenEndStatement},
	"{% default %}":                {tokenStartStatement, tokenDefault, tokenEndStatement},
	"{% fallthrough %}":            {tokenStartStatement, tokenFallthrough, tokenEndStatement},
	"{% switch a := 5; a %}":       {tokenStartStatement, tokenSwitch, tokenIdentifier, tokenDeclaration, tokenInt, tokenSemicolon, tokenIdentifier, tokenEndStatement},
	"{% switch a %}":               {tokenStartStatement, tokenSwitch, tokenIdentifier, tokenEndStatement},
	"{% switch a.(type) %}":        {tokenStartStatement, tokenSwitch, tokenIdentifier, tokenPeriod, tokenLeftParenthesis, tokenType, tokenRightParenthesis, tokenEndStatement},
	"{% switch v := a.(type) %}":   {tokenStartStatement, tokenSwitch, tokenIdentifier, tokenDeclaration, tokenIdentifier, tokenPeriod, tokenLeftParenthesis, tokenType, tokenRightParenthesis, tokenEndStatement},
	"{% else %}":                   {tokenStartStatement, tokenElse, tokenEndStatement},
	"{% extends \"\" %}":           {tokenStartStatement, tokenExtends, tokenInterpretedString, tokenEndStatement},
	"{% macro a %}":                {tokenStartStatement, tokenMacro, tokenIdentifier, tokenEndStatement},
	"{% macro a(b) %}":             {tokenStartStatement, tokenMacro, tokenIdentifier, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndStatement},
	"{% macro a(b...) %}":          {tokenStartStatement, tokenMacro, tokenIdentifier, tokenLeftParenthesis, tokenIdentifier, tokenEllipses, tokenRightParenthesis, tokenEndStatement},
	"{% include \"\" %}":           {tokenStartStatement, tokenInclude, tokenInterpretedString, tokenEndStatement},
	"{# comment #}":                {tokenComment},
	`a{{b}}c`:                      {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}c`:                       {tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}\n`:                      {tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	`{{a}}{{b}}`:                   {tokenStartValue, tokenIdentifier, tokenEndValue, tokenStartValue, tokenIdentifier, tokenEndValue},
	"<script></script>":            {tokenText},
	"<style></style>":              {tokenText},
	"<script>{{a}}</script>":       {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"<style>{{a}}</style>":         {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"<a class=\"{{c}}\"></a>":      {tokenText, tokenStartValue, tokenIdentifier, tokenEndValue, tokenText},
	"{{ _ }}":                      {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ __ }}":                     {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ _a }}":                     {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ a5 }}":                     {tokenStartValue, tokenIdentifier, tokenEndValue},
	"{{ 3 }}":                      {tokenStartValue, tokenInt, tokenEndValue},
	"{{ -3 }}":                     {tokenStartValue, tokenSubtraction, tokenInt, tokenEndValue},
	"{{ +3 }}":                     {tokenStartValue, tokenAddition, tokenInt, tokenEndValue},
	"{{ 0 }}":                      {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 2147483647 }}":             {tokenStartValue, tokenInt, tokenEndValue},
	"{{ -2147483648 }}":            {tokenStartValue, tokenSubtraction, tokenInt, tokenEndValue},
	"{{ .0 }}":                     {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 0. }}":                     {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 0.0 }}":                    {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 2147483647.2147483647 }}":  {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ -2147483647.2147483647 }}": {tokenStartValue, tokenSubtraction, tokenFloat, tokenEndValue},
	"{{ 2147483647.2147483647214748364721474836472 }}": {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 1 }}":                {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0.1 }}":              {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 1.1 }}":              {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ .1 }}":               {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 3903 }}":             {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 3903.902634 }}":      {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 0e0 }}":              {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 0E0 }}":              {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 12.90e23 }}":         {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ .1923783E91 }}":      {tokenStartValue, tokenFloat, tokenEndValue},
	"{{ 01 }}":               {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 01234567 }}":         {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0o1234567 }}":        {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0O1234567 }}":        {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0x12345679ABCDEF }}": {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 0X12345679ABCDEF }}": {tokenStartValue, tokenInt, tokenEndValue},
	"{{ 'j' }}":              {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\n' }}":            {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\106' }}":          {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\x6a' }}":          {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\x6A' }}":          {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\377' }}":          {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\u006A' }}":        {tokenStartValue, tokenRune, tokenEndValue},
	"{{ '\\U0000006A' }}":    {tokenStartValue, tokenRune, tokenEndValue},
	"{{ a + b }}":            {tokenStartValue, tokenIdentifier, tokenAddition, tokenIdentifier, tokenEndValue},
	"{{ a - b }}":            {tokenStartValue, tokenIdentifier, tokenSubtraction, tokenIdentifier, tokenEndValue},
	"{{ a * b }}":            {tokenStartValue, tokenIdentifier, tokenMultiplication, tokenIdentifier, tokenEndValue},
	"{{ a / b }}":            {tokenStartValue, tokenIdentifier, tokenDivision, tokenIdentifier, tokenEndValue},
	"{{ a % b }}":            {tokenStartValue, tokenIdentifier, tokenModulo, tokenIdentifier, tokenEndValue},
	"{{ ( a ) }}":            {tokenStartValue, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndValue},
	"{{ a == b }}":           {tokenStartValue, tokenIdentifier, tokenEqual, tokenIdentifier, tokenEndValue},
	"{{ a != b }}":           {tokenStartValue, tokenIdentifier, tokenNotEqual, tokenIdentifier, tokenEndValue},
	"{{ a && b }}":           {tokenStartValue, tokenIdentifier, tokenAndAnd, tokenIdentifier, tokenEndValue},
	"{{ a || b }}":           {tokenStartValue, tokenIdentifier, tokenOrOr, tokenIdentifier, tokenEndValue},
	"{{ a < b }}":            {tokenStartValue, tokenIdentifier, tokenLess, tokenIdentifier, tokenEndValue},
	"{{ a <= b }}":           {tokenStartValue, tokenIdentifier, tokenLessOrEqual, tokenIdentifier, tokenEndValue},
	"{{ a > b }}":            {tokenStartValue, tokenIdentifier, tokenGreater, tokenIdentifier, tokenEndValue},
	"{{ a >= b }}":           {tokenStartValue, tokenIdentifier, tokenGreaterOrEqual, tokenIdentifier, tokenEndValue},
	"{{ !a }}":               {tokenStartValue, tokenNot, tokenIdentifier, tokenEndValue},
	"{{ a[5] }}":             {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInt, tokenRightBrackets, tokenEndValue},
	"{{ &a }}":               {tokenStartValue, tokenAnd, tokenIdentifier, tokenEndValue},
	"{{ 4 + &f(2) }}": {tokenStartValue, tokenInt, tokenAddition,
		tokenAnd, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenRightParenthesis, tokenEndValue},
	"{{ *a }}":              {tokenStartValue, tokenMultiplication, tokenIdentifier, tokenEndValue},
	"{{ []*int{} }}":        {tokenStartValue, tokenLeftBrackets, tokenRightBrackets, tokenMultiplication, tokenIdentifier, tokenLeftBraces, tokenRightBraces, tokenEndValue},
	"{{ a[\"5\"] }}":        {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInterpretedString, tokenRightBrackets, tokenEndValue},
	"{{ a[:] }}":            {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenRightBrackets, tokenEndValue},
	"{{ a[:8] }}":           {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenColon, tokenInt, tokenRightBrackets, tokenEndValue},
	"{{ a[3:] }}":           {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInt, tokenColon, tokenRightBrackets, tokenEndValue},
	"{{ a[3:8] }}":          {tokenStartValue, tokenIdentifier, tokenLeftBrackets, tokenInt, tokenColon, tokenInt, tokenRightBrackets, tokenEndValue},
	"{{ a() }}":             {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenEndValue},
	"{{ a(1) }}":            {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenRightParenthesis, tokenEndValue},
	"{{ a(1,2) }}":          {tokenStartValue, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenComma, tokenInt, tokenRightParenthesis, tokenEndValue},
	"{{ a.b }}":             {tokenStartValue, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenEndValue},
	"{{ \"\" }}":            {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u09AF\" }}":     {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u09af\" }}":     {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\U00008a9e\" }}": {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\U0010FFFF\" }}": {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\t\" }}":         {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ \"\\u3C2E\\\"\" }}": {tokenStartValue, tokenInterpretedString, tokenEndValue},
	"{{ `` }}":              {tokenStartValue, tokenRawString, tokenEndValue},
	"{{ `\\t` }}":           {tokenStartValue, tokenRawString, tokenEndValue},
	"{{ ( 1 + 2 ) * 3 }}": {tokenStartValue, tokenLeftParenthesis, tokenInt, tokenAddition, tokenInt, tokenRightParenthesis,
		tokenMultiplication, tokenInt, tokenEndValue},
	"{{ map{} }}":       {tokenStartValue, tokenMap, tokenLeftBraces, tokenRightBraces, tokenEndValue},
	"{{ map{`a`: 6} }}": {tokenStartValue, tokenMap, tokenLeftBraces, tokenRawString, tokenColon, tokenInt, tokenSemicolon, tokenRightBraces, tokenEndValue},
	"{{ interface{} }}": {tokenStartValue, tokenInterface, tokenLeftBraces, tokenRightBraces, tokenEndValue},
}

var typeTestsGoContext = map[string][]tokenTyp{
	``:                {},
	"a := 3":          {tokenIdentifier, tokenDeclaration, tokenInt, tokenSemicolon},
	"// a comment\n":  {},
	`// a comment`:    {},
	`3`:               {tokenInt, tokenSemicolon},
	`3 // comment`:    {tokenInt, tokenSemicolon},
	"3 // comment\n4": {tokenInt, tokenSemicolon, tokenInt, tokenSemicolon},
	"// a comment\na := 7\n// another comment\n": {tokenIdentifier, tokenDeclaration, tokenInt, tokenSemicolon},
	`/* a comment */`:                     {},
	"/* a comment \n another line */":     {},
	"f()/* a comment\nanother line */g()": {tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon},
	`a = /* comment */ b`:                 {tokenIdentifier, tokenSimpleAssignment, tokenIdentifier, tokenSemicolon},
	"var a":                               {tokenVar, tokenIdentifier, tokenSemicolon},
	"var a int = 5":                       {tokenVar, tokenIdentifier, tokenIdentifier, tokenSimpleAssignment, tokenInt, tokenSemicolon},
	"const b, c = 8, 10":                  {tokenConst, tokenIdentifier, tokenComma, tokenIdentifier, tokenSimpleAssignment, tokenInt, tokenComma, tokenInt, tokenSemicolon},
	"type Int int":                        {tokenType, tokenIdentifier, tokenIdentifier, tokenSemicolon},
	"type stringSlice []string":           {tokenType, tokenIdentifier, tokenLeftBrackets, tokenRightBrackets, tokenIdentifier, tokenSemicolon},
	"struct { A, B T1 ; C, D T2 }":        {tokenStruct, tokenLeftBraces, tokenIdentifier, tokenComma, tokenIdentifier, tokenIdentifier, tokenSemicolon, tokenIdentifier, tokenComma, tokenIdentifier, tokenIdentifier, tokenSemicolon, tokenRightBraces, tokenSemicolon},
}

var contextTests = map[ast.Context]map[string][]ast.Context{
	ast.ContextText: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextText, ast.ContextText, ast.ContextText},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText, ast.ContextText},
	},
	ast.ContextHTML: {
		`a`:                                            {ast.ContextText},
		`{{a}}`:                                        {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		"<script></script>":                            {ast.ContextText},
		"<style></style>":                              {ast.ContextText},
		"<script>s{{a}}</script>{{a}}":                 {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		"<style>s{{a}}</style>{{a}}":                   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style>s{{include "a"}}t</style>`:             {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<script>s{{include "a"}}t</script>`:           {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText},
		`<style a="{{b}}"></style>`:                    {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<style a="b">{{1}}</style>`:                   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<script a="{{b}}"></script>`:                  {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<script a="b">{{1}}</script>`:                 {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText},
		`<![CDATA[<script>{{1}}</script>]]>`:           {ast.ContextText},
		`a<![CDATA[<script>{{1}}</script>]]>{{2}}`:     {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<div {{ attr }}>`:                             {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div {{ attr }} {{ attr }}>`:                  {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div{{ attr }}>`:                              {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div {{ attr }}="45">`:                        {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div "{{ v }}">`:                              {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<a href="">`:                                  {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<A Href="">`:                                  {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<a href=''>`:                                  {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<a href="{{ u }}">`:                           {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<a href={{ u }}>`:                             {ast.ContextText, ast.ContextUnquotedAttribute, ast.ContextUnquotedAttribute, ast.ContextUnquotedAttribute, ast.ContextUnquotedAttribute, ast.ContextUnquotedAttribute, ast.ContextText},
		`<a href="a{{ p }}">`:                          {ast.ContextText, ast.ContextAttribute, ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<a href="{% if a %}b{% end %}">`:              {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<a class="{{ a }}">`:                          {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<input type="text" disabled class="{{ a }}">`: {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		`<input type="text" data-value="{{ a }}">`:     {ast.ContextText, ast.ContextAttribute, ast.ContextAttribute, ast.ContextAttribute, ast.ContextText},
		"<style>s{{a}}t</style>{{a}}":                  {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style>{{a}}"{{a}}"</style>`:                  {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText},
		`<style>"{{a}}"{{a}}</style>`:                  {ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<style>{{a}}'{{a}}'</style>`:                  {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText},
		`<style>'{{a}}'{{a}}</style>`:                  {ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<style>'</style>'{{a}}</style>`:               {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<style>"</style>"{{a}}</style>`:               {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		"<script>s{{a}}t</script>{{a}}":                {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script>{{a}}"{{a}}"</script>`:                {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextText},
		`<script>"{{a}}"{{a}}</script>`:                {ast.ContextText, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText},
		`<script>{{a}}'{{a}}'</script>`:                {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextText},
		`<script>'{{a}}'{{a}}</script>`:                {ast.ContextText, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText},
		`<script>'</script>'{{a}}</script>`:            {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<script>"</script>"{{a}}</script>`:            {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<script async></script>{{ "a" }}`:             {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
	},
	ast.ContextCSS: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextCSS, ast.ContextCSS, ast.ContextCSS},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS},
	},
	ast.ContextCSSString: {
		`a`:       {ast.ContextText},
		`a{{a}}a`: {ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText},
	},
	ast.ContextJavaScript: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextText, ast.ContextJavaScript, ast.ContextJavaScript, ast.ContextJavaScript},
	},
	ast.ContextJavaScriptString: {
		`a`:       {ast.ContextText},
		`a{{a}}a`: {ast.ContextText, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextJavaScriptString, ast.ContextText},
	},
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
	{"<b c=\n{{\na}}>", []ast.Position{
		{1, 1, 0, 5}, {2, 1, 6, 7}, {3, 1, 9, 9}, {3, 2, 10, 11}, {3, 4, 12, 12}}},
	{"{{a}}\n<b\nc=\"{{a}}\">\n{{a}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}, {1, 6, 5, 11},
		{3, 4, 12, 13}, {3, 6, 14, 14}, {3, 7, 15, 16}, {3, 9, 17, 19},
		{4, 1, 20, 21}, {4, 3, 22, 22}, {4, 4, 23, 24}}},
	{"a<![CDATA[a\nb]]>b{{a}}", []ast.Position{
		{1, 1, 0, 16}, {2, 6, 17, 18}, {2, 8, 19, 19}, {2, 9, 20, 21}}},
	{"a{# a #}b", []ast.Position{
		{1, 1, 0, 0}, {1, 2, 1, 7}, {1, 9, 8, 8}}},
	{"a{# 本 #}b", []ast.Position{
		{1, 1, 0, 0}, {1, 2, 1, 9}, {1, 9, 10, 10}}},
}

var scanTagTests = []struct {
	src    string
	tag    string
	p      int
	line   int
	column int
}{
	{"a ", "a", 1, 1, 2},
	{"img\n", "img", 3, 1, 4},
	{"href\t", "href", 4, 1, 5},
	{"a", "a", 1, 1, 2},
	{"a5 ", "a5", 2, 1, 3},
	{" ", "", 0, 1, 1},
	{"\n", "", 0, 1, 1},
	{"5 ", "", 1, 1, 2},
	{" a", "", 0, 1, 1},
}

var scanAttributeTests = []struct {
	src    string
	attr   string
	quote  byte
	p      int
	line   int
	column int
}{
	{"href=\"h", "href", '"', 5, 1, 6},
	{"href='h", "href", '\'', 5, 1, 6},
	{"src = \"h", "src", '"', 6, 1, 7},
	{"src\n= \"h", "src", '"', 6, 2, 3},
	{"src =\n\"h", "src", '"', 6, 2, 1},
	{"src\t\t=\n\"h", "src", '"', 7, 2, 1},
	{"a='h", "a", '\'', 2, 1, 3},
	{"src=h", "src", 0, 4, 1, 5},
	{"src=\n\th", "src", 0, 6, 2, 2},
	{"src=/a/b", "src", 0, 4, 1, 5},
	{"src=>", "", 0, 4, 1, 5},
	{"src=/>", "src", 0, 4, 1, 5},
	{"src", "", 0, 3, 1, 4},
	{"src=", "", 0, 4, 1, 5},
	{"src ", "", 0, 4, 1, 5},
	{"s5c='h", "s5c", '\'', 4, 1, 5},
	{"本='h", "本", '\'', 4, 1, 3},
	{"a本-€本b='h", "a本-€本b", '\'', 13, 1, 8},
	{"5c=\"", "5c", '"', 3, 1, 4},
}

func testLexerTypes(t *testing.T, test map[string][]tokenTyp, ctx ast.Context) {
TYPES:
	for source, types := range test {
		var lex = newLexer([]byte(source), ctx)
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
			t.Errorf("source: %q, less types\n", source)
		}
	}
}

func TestLexerTypes(t *testing.T) {
	testLexerTypes(t, typeTests, ast.ContextText)
}

func TestLexerTypesGoContext(t *testing.T) {
	testLexerTypes(t, typeTestsGoContext, ast.ContextGo)
}

func TestLexerContexts(t *testing.T) {
CONTEXTS:
	for ctx, tests := range contextTests {
		for source, contexts := range tests {
			var lex = newLexer([]byte(source), ctx)
			var i int
			for tok := range lex.tokens {
				if tok.typ == tokenEOF {
					break
				}
				if i >= len(contexts) {
					t.Errorf("source: %q, missing context in test\n", source)
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
				t.Errorf("source: %q, less contexts\n", source)
			}
		}
	}
}

func TestPositions(t *testing.T) {
	for _, test := range positionTests {
		var lex = newLexer([]byte(test.src), ast.ContextHTML)
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
			t.Errorf("source: %q, less lines\n", test.src)
		}
	}
}

func TestLexerReadTag(t *testing.T) {
	for _, test := range scanTagTests {
		src := []byte(test.src)
		var l = &lexer{
			text:   src,
			src:    src,
			line:   1,
			column: 1,
			ctx:    ast.ContextHTML,
		}
		tag, p := l.scanTag(0)
		if tag != test.tag {
			t.Errorf("source: %q, unexpected tag %q, expecting %q\n", test.src, tag, test.tag)
		}
		if p != test.p {
			t.Errorf("source: %q, unexpected position %d, expecting %d\n", test.src, p, test.p)
		}
		if l.line != test.line {
			t.Errorf("source: %q, unexpected line %d, expecting %d\n", test.src, l.line, test.line)
		}
		if l.column != test.column {
			t.Errorf("source: %q, unexpected column %d, expecting %d\n", test.src, l.column, test.column)
		}
	}
}

func TestLexerReadAttribute(t *testing.T) {
	for _, test := range scanAttributeTests {
		src := []byte(test.src)
		var l = &lexer{
			text:   src,
			src:    src,
			line:   1,
			column: 1,
			ctx:    ast.ContextHTML,
		}
		attr, p := l.scanAttribute(0)
		if attr != test.attr {
			t.Errorf("source: %q, unexpected attribute %q, expecting %q\n", test.src, attr, test.attr)
		}
		if attr != "" && test.quote != 0 && l.src[p] != test.quote {
			t.Errorf("source: %q, unexpected quote %q, expecting %q\n", test.src, string(l.src[p]), string(test.quote))
		}
		if p != test.p {
			t.Errorf("source: %q, unexpected position %d, expecting %d\n", test.src, p, test.p)
		}
		if l.line != test.line {
			t.Errorf("source: %q, unexpected line %d, expecting %d\n", test.src, l.line, test.line)
		}
		if l.column != test.column {
			t.Errorf("source: %q, unexpected column %d, expecting %d\n", test.src, l.column, test.column)
		}
	}
}
