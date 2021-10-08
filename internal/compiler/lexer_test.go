// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"strings"
	"testing"

	"github.com/open2b/scriggo/ast"
)

var typeTestsText = map[string][]tokenTyp{
	``:                             {},
	`a`:                            {tokenText},
	`{`:                            {tokenText},
	`}`:                            {tokenText},
	`{{a}}`:                        {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	`{{ a }}`:                      {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"{{\ta\n}}":                    {tokenLeftBraces, tokenIdentifier, tokenSemicolon, tokenRightBraces},
	"{{\na\t}}":                    {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"{{\na;\t}}":                   {tokenLeftBraces, tokenIdentifier, tokenSemicolon, tokenRightBraces},
	"{% a := 1 %}":                 {tokenStartStatement, tokenIdentifier, tokenDeclaration, tokenInt, tokenEndStatement},
	"{% a = 2 %}":                  {tokenStartStatement, tokenIdentifier, tokenSimpleAssignment, tokenInt, tokenEndStatement},
	"{% a += 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenAdditionAssignment, tokenInt, tokenEndStatement},
	"{% a -= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenSubtractionAssignment, tokenInt, tokenEndStatement},
	"{% a *= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenMultiplicationAssignment, tokenInt, tokenEndStatement},
	"{% a /= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenDivisionAssignment, tokenInt, tokenEndStatement},
	"{% a %= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenModuloAssignment, tokenInt, tokenEndStatement},
	"{% a &= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenAndAssignment, tokenInt, tokenEndStatement},
	"{% a |= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenOrAssignment, tokenInt, tokenEndStatement},
	"{% a ^= 3 %}":                 {tokenStartStatement, tokenIdentifier, tokenXorAssignment, tokenInt, tokenEndStatement},
	"{% a &^= 3 %}":                {tokenStartStatement, tokenIdentifier, tokenAndNotAssignment, tokenInt, tokenEndStatement},
	"{% a <<= 3 %}":                {tokenStartStatement, tokenIdentifier, tokenLeftShiftAssignment, tokenInt, tokenEndStatement},
	"{% a >>= 3 %}":                {tokenStartStatement, tokenIdentifier, tokenRightShiftAssignment, tokenInt, tokenEndStatement},
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
	"{% if not a %}":               {tokenStartStatement, tokenIf, tokenExtendedNot, tokenIdentifier, tokenEndStatement},
	"{% if not 10 + 3 %}":          {tokenStartStatement, tokenIf, tokenExtendedNot, tokenInt, tokenAddition, tokenInt, tokenEndStatement},
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
	"{% macro a html %}":           {tokenStartStatement, tokenMacro, tokenIdentifier, tokenIdentifier, tokenEndStatement},
	"{% macro a(b) %}":             {tokenStartStatement, tokenMacro, tokenIdentifier, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenEndStatement},
	"{% macro a(b) markdown %}":    {tokenStartStatement, tokenMacro, tokenIdentifier, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenIdentifier, tokenEndStatement},
	"{% macro a(b...) %}":          {tokenStartStatement, tokenMacro, tokenIdentifier, tokenLeftParenthesis, tokenIdentifier, tokenEllipsis, tokenRightParenthesis, tokenEndStatement},
	"{{ render \"\" }}":            {tokenLeftBraces, tokenRender, tokenInterpretedString, tokenRightBraces},
	"{% show(5) %}":                {tokenStartStatement, tokenShow, tokenLeftParenthesis, tokenInt, tokenRightParenthesis, tokenEndStatement},
	"{% show `a`, 7, true %}":      {tokenStartStatement, tokenShow, tokenRawString, tokenComma, tokenInt, tokenComma, tokenIdentifier, tokenEndStatement},
	"{%% a := 1  %%}":              {tokenStartStatements, tokenIdentifier, tokenDeclaration, tokenInt, tokenSemicolon, tokenEndStatements},
	"{%% var a int;\na = 1; %%}":   {tokenStartStatements, tokenVar, tokenIdentifier, tokenIdentifier, tokenSemicolon, tokenIdentifier, tokenSimpleAssignment, tokenInt, tokenSemicolon, tokenEndStatements},
	"{# comment #}":                {tokenComment},
	"{# nested {# comment #} #}":   {tokenComment},
	`a{{b}}c`:                      {tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText},
	`{{a}}c`:                       {tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText},
	`{{a}}\n`:                      {tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText},
	`{{a}}{{b}}`:                   {tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"<script></script>":            {tokenText},
	"<style></style>":              {tokenText},
	"<script>{{a}}</script>":       {tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText},
	"<style>{{a}}</style>":         {tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText},
	"<a class=\"{{c}}\"></a>":      {tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText},
	"{{ _ }}":                      {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"{{ __ }}":                     {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"{{ _a }}":                     {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"{{ a5 }}":                     {tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"{{ 3 }}":                      {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ -3 }}":                     {tokenLeftBraces, tokenSubtraction, tokenInt, tokenRightBraces},
	"{{ +3 }}":                     {tokenLeftBraces, tokenAddition, tokenInt, tokenRightBraces},
	"{{ 0 }}":                      {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 2147483647 }}":             {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ -2147483648 }}":            {tokenLeftBraces, tokenSubtraction, tokenInt, tokenRightBraces},
	"{{ .0 }}":                     {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0. }}":                     {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0.0 }}":                    {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 2147483647.2147483647 }}":  {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ -2147483647.2147483647 }}": {tokenLeftBraces, tokenSubtraction, tokenFloat, tokenRightBraces},
	"{{ 2147483647.2147483647214748364721474836472 }}": {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 1 }}":                {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0.1 }}":              {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 1.1 }}":              {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ .1 }}":               {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 3903 }}":             {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 3903.902634 }}":      {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0e0 }}":              {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0E0 }}":              {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 12.90e23 }}":         {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ .1923783E91 }}":      {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0x7f7ffffe }}":       {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0.E1 }}":             {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 01 }}":               {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 01234567 }}":         {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0o1234567 }}":        {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0O1234567 }}":        {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0x12345679ABCDEF }}": {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0X12345679ABCDEF }}": {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0b01 }}":             {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0B01 }}":             {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0o51701 }}":          {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0O51701 }}":          {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0x1b6F.c2Ap15 }}":    {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0.8e-45 }}":          {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 1_2 }}":              {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 1_23_456_789 }}":     {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 1_2.3_4 }}":          {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 1_2.3_4e5_6 }}":      {tokenLeftBraces, tokenFloat, tokenRightBraces},
	"{{ 0_0 }}":              {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0_123_456 }}":        {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0x123_456 }}":        {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0b10_01_1 }}":        {tokenLeftBraces, tokenInt, tokenRightBraces},
	"{{ 0i }}":               {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 5i }}":               {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 0.6i }}":             {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 0.6e2i }}":           {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 066i }}":             {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 069i }}":             {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 084i }}":             {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 0.i }}":              {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ .0i }}":              {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 0xABC.Ap-4i }}":      {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 0xABC.Ap+4i }}":      {tokenLeftBraces, tokenImaginary, tokenRightBraces},
	"{{ 'j' }}":              {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\n' }}":            {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\106' }}":          {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\x6a' }}":          {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\x6A' }}":          {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\377' }}":          {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\u006A' }}":        {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ '\\U0000006A' }}":    {tokenLeftBraces, tokenRune, tokenRightBraces},
	"{{ a + b }}":            {tokenLeftBraces, tokenIdentifier, tokenAddition, tokenIdentifier, tokenRightBraces},
	"{{ a - b }}":            {tokenLeftBraces, tokenIdentifier, tokenSubtraction, tokenIdentifier, tokenRightBraces},
	"{{ a * b }}":            {tokenLeftBraces, tokenIdentifier, tokenMultiplication, tokenIdentifier, tokenRightBraces},
	"{{ a / b }}":            {tokenLeftBraces, tokenIdentifier, tokenDivision, tokenIdentifier, tokenRightBraces},
	"{{ a % b }}":            {tokenLeftBraces, tokenIdentifier, tokenModulo, tokenIdentifier, tokenRightBraces},
	"{{ ( a ) }}":            {tokenLeftBraces, tokenLeftParenthesis, tokenIdentifier, tokenRightParenthesis, tokenRightBraces},
	"{{ a == b }}":           {tokenLeftBraces, tokenIdentifier, tokenEqual, tokenIdentifier, tokenRightBraces},
	"{{ a != b }}":           {tokenLeftBraces, tokenIdentifier, tokenNotEqual, tokenIdentifier, tokenRightBraces},
	"{{ a && b }}":           {tokenLeftBraces, tokenIdentifier, tokenAnd, tokenIdentifier, tokenRightBraces},
	"{{ a || b }}":           {tokenLeftBraces, tokenIdentifier, tokenOr, tokenIdentifier, tokenRightBraces},
	"{{ a < b }}":            {tokenLeftBraces, tokenIdentifier, tokenLess, tokenIdentifier, tokenRightBraces},
	"{{ a <= b }}":           {tokenLeftBraces, tokenIdentifier, tokenLessOrEqual, tokenIdentifier, tokenRightBraces},
	"{{ a > b }}":            {tokenLeftBraces, tokenIdentifier, tokenGreater, tokenIdentifier, tokenRightBraces},
	"{{ a >= b }}":           {tokenLeftBraces, tokenIdentifier, tokenGreaterOrEqual, tokenIdentifier, tokenRightBraces},
	"{{ !a }}":               {tokenLeftBraces, tokenNot, tokenIdentifier, tokenRightBraces},
	"{{ a[5] }}":             {tokenLeftBraces, tokenIdentifier, tokenLeftBracket, tokenInt, tokenRightBracket, tokenRightBraces},
	"{{ &a }}":               {tokenLeftBraces, tokenAmpersand, tokenIdentifier, tokenRightBraces},
	"{{ 4 + &f(2) }}": {tokenLeftBraces, tokenInt, tokenAddition,
		tokenAmpersand, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenRightParenthesis, tokenRightBraces},
	"{{ *a }}":              {tokenLeftBraces, tokenMultiplication, tokenIdentifier, tokenRightBraces},
	"{{ []*int{} }}":        {tokenLeftBraces, tokenLeftBracket, tokenRightBracket, tokenMultiplication, tokenIdentifier, tokenLeftBrace, tokenRightBrace, tokenRightBraces},
	"{{ $a }}":              {tokenLeftBraces, tokenDollar, tokenIdentifier, tokenRightBraces},
	"{{ a[\"5\"] }}":        {tokenLeftBraces, tokenIdentifier, tokenLeftBracket, tokenInterpretedString, tokenRightBracket, tokenRightBraces},
	"{{ a[:] }}":            {tokenLeftBraces, tokenIdentifier, tokenLeftBracket, tokenColon, tokenRightBracket, tokenRightBraces},
	"{{ a[:8] }}":           {tokenLeftBraces, tokenIdentifier, tokenLeftBracket, tokenColon, tokenInt, tokenRightBracket, tokenRightBraces},
	"{{ a[3:] }}":           {tokenLeftBraces, tokenIdentifier, tokenLeftBracket, tokenInt, tokenColon, tokenRightBracket, tokenRightBraces},
	"{{ a[3:8] }}":          {tokenLeftBraces, tokenIdentifier, tokenLeftBracket, tokenInt, tokenColon, tokenInt, tokenRightBracket, tokenRightBraces},
	"{{ a() }}":             {tokenLeftBraces, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenRightBraces},
	"{{ a(1) }}":            {tokenLeftBraces, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenRightParenthesis, tokenRightBraces},
	"{{ a(1,2) }}":          {tokenLeftBraces, tokenIdentifier, tokenLeftParenthesis, tokenInt, tokenComma, tokenInt, tokenRightParenthesis, tokenRightBraces},
	"{{ a.b }}":             {tokenLeftBraces, tokenIdentifier, tokenPeriod, tokenIdentifier, tokenRightBraces},
	"{{ \"\" }}":            {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ \"\\u09AF\" }}":     {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ \"\\u09af\" }}":     {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ \"\\U00008a9e\" }}": {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ \"\\U0010FFFF\" }}": {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ \"\\t\" }}":         {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ \"\\u3C2E\\\"\" }}": {tokenLeftBraces, tokenInterpretedString, tokenRightBraces},
	"{{ `` }}":              {tokenLeftBraces, tokenRawString, tokenRightBraces},
	"{{ `\\t` }}":           {tokenLeftBraces, tokenRawString, tokenRightBraces},
	"{{ ( 1 + 2 ) * 3 }}": {tokenLeftBraces, tokenLeftParenthesis, tokenInt, tokenAddition, tokenInt, tokenRightParenthesis,
		tokenMultiplication, tokenInt, tokenRightBraces},
	"{{ map{} }}":                   {tokenLeftBraces, tokenMap, tokenLeftBrace, tokenRightBrace, tokenRightBraces},
	"{{ map{`a`: 6} }}":             {tokenLeftBraces, tokenMap, tokenLeftBrace, tokenRawString, tokenColon, tokenInt, tokenRightBrace, tokenRightBraces},
	"{{ interface{} }}":             {tokenLeftBraces, tokenInterface, tokenLeftBrace, tokenRightBrace, tokenRightBraces},
	"{{ a and b }}":                 {tokenLeftBraces, tokenIdentifier, tokenExtendedAnd, tokenIdentifier, tokenRightBraces},
	"{{ a or b }}":                  {tokenLeftBraces, tokenIdentifier, tokenExtendedOr, tokenIdentifier, tokenRightBraces},
	"{{ a or not b }}":              {tokenLeftBraces, tokenIdentifier, tokenExtendedOr, tokenExtendedNot, tokenIdentifier, tokenRightBraces},
	"{{ a contains b }}":            {tokenLeftBraces, tokenIdentifier, tokenContains, tokenIdentifier, tokenRightBraces},
	"{{ a not contains b }}":        {tokenLeftBraces, tokenIdentifier, tokenExtendedNot, tokenContains, tokenIdentifier, tokenRightBraces},
	"{% raw %}t{% end %}":           {tokenStartStatement, tokenRaw, tokenEndStatement, tokenText, tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% raw %}{% if {% end %}":      {tokenStartStatement, tokenRaw, tokenEndStatement, tokenText, tokenStartStatement, tokenEnd, tokenEndStatement},
	"{% raw %} if %}{% end %}":      {tokenStartStatement, tokenRaw, tokenEndStatement, tokenText, tokenStartStatement, tokenEnd, tokenEndStatement},
	"{{ a default b }}":             {tokenLeftBraces, tokenIdentifier, tokenDefault, tokenIdentifier, tokenRightBraces},
	"{% a = itea; using %}":         {tokenStartStatement, tokenIdentifier, tokenSimpleAssignment, tokenIdentifier, tokenSemicolon, tokenUsing, tokenEndStatement},
	"{% a = itea(); using macro %}": {tokenStartStatement, tokenIdentifier, tokenSimpleAssignment, tokenIdentifier, tokenLeftParenthesis, tokenRightParenthesis, tokenSemicolon, tokenUsing, tokenMacro, tokenEndStatement},
	"<a {% if a %}{% end %}>":       {tokenText, tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement, tokenStartStatement, tokenEnd, tokenEndStatement, tokenText},
	"<a {% if a %}b{% end %}>":      {tokenText, tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement, tokenText, tokenStartStatement, tokenEnd, tokenEndStatement, tokenText},
	"<a {% if a %}b=\"\"{% end %}>": {tokenText, tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement, tokenText, tokenStartStatement, tokenEnd, tokenEndStatement, tokenText},
	"<a {% if a %}b=''{% end %}>":   {tokenText, tokenStartStatement, tokenIf, tokenIdentifier, tokenEndStatement, tokenText, tokenStartStatement, tokenEnd, tokenEndStatement, tokenText},
	"#! /usr/bin/scriggo\n{{ a }}":  {tokenShebangLine, tokenLeftBraces, tokenIdentifier, tokenRightBraces},
	"#! /usr/bin/scriggo":           {tokenShebangLine},
	"#! /usr/bin/scriggo\n":         {tokenShebangLine},
}

var tagWithURLTypes = []tokenTyp{tokenText, tokenStartURL, tokenText, tokenEndURL, tokenText}

var typeTestsHTML = map[string][]tokenTyp{
	`<form action="u">`:       tagWithURLTypes,
	`<blockquote cite="u">`:   tagWithURLTypes,
	`<del cite="u">`:          tagWithURLTypes,
	`<ins cite="u">`:          tagWithURLTypes,
	`<q cite="u">`:            tagWithURLTypes,
	`<object data="u">`:       tagWithURLTypes,
	`<button formaction="u">`: tagWithURLTypes,
	`<input formaction="u">`:  tagWithURLTypes,
	`<a href="u">`:            tagWithURLTypes,
	`<area href="u">`:         tagWithURLTypes,
	`<link href="u">`:         tagWithURLTypes,
	`<base href="u">`:         tagWithURLTypes,
	`<img longdesc="u">`:      tagWithURLTypes,
	`<html manifest="u">`:     tagWithURLTypes,
	`<video poster="u">`:      tagWithURLTypes,
	`<audio src="u">`:         tagWithURLTypes,
	`<embed src="u">`:         tagWithURLTypes,
	`<iframe src="u">`:        tagWithURLTypes,
	`<img src="u">`:           tagWithURLTypes,
	`<input src="u">`:         tagWithURLTypes,
	`<script src="u">`:        tagWithURLTypes,
	`<source src="u">`:        tagWithURLTypes,
	`<track src="u">`:         tagWithURLTypes,
	`<video src="u">`:         tagWithURLTypes,
	`<img srcset="u">`:        tagWithURLTypes,
	`<a data-href="u">`:       tagWithURLTypes,
	`<input data-src="u">`:    tagWithURLTypes,
	`<a my:href="u">`:         tagWithURLTypes,
	`<tag xmlns="u">`:         tagWithURLTypes,
	`<tag xmlns:h="u">`:       tagWithURLTypes,
	`<tag data-src="u">`:      tagWithURLTypes,
	`<tag data-imageurl="u">`: tagWithURLTypes,
	`<tag data-uri-x="u">`:    tagWithURLTypes,
	`<tag data-x-src-x="u">`:  tagWithURLTypes,
}

var typeTestsMarkdown = map[string][]tokenTyp{
	` https://`:                       {tokenText, tokenStartURL, tokenText, tokenEndURL},
	` https://site.com/`:              {tokenText, tokenStartURL, tokenText, tokenEndURL},
	`https://{{ domain }}`:            {tokenStartURL, tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenEndURL},
	`https://site.com/?a={{ v }}`:     {tokenStartURL, tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenEndURL},
	` https://site.com/?a={{ v }} `:   {tokenText, tokenStartURL, tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenEndURL, tokenText},
	`https://{{ domain }}/?a={{ v }}`: {tokenStartURL, tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenText, tokenLeftBraces, tokenIdentifier, tokenRightBraces, tokenEndURL},
}

var typeTestsGo = map[string][]tokenTyp{
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
	"type stringSlice []string":           {tokenType, tokenIdentifier, tokenLeftBracket, tokenRightBracket, tokenIdentifier, tokenSemicolon},
	"struct { A, B T1 ; C, D T2 }":        {tokenStruct, tokenLeftBrace, tokenIdentifier, tokenComma, tokenIdentifier, tokenIdentifier, tokenSemicolon, tokenIdentifier, tokenComma, tokenIdentifier, tokenIdentifier, tokenRightBrace, tokenSemicolon},
	"#! /usr/bin/scriggo\nvar a":          {tokenShebangLine, tokenVar, tokenIdentifier, tokenSemicolon},
	"#! /usr/bin/scriggo":                 {tokenShebangLine},
	"#! /usr/bin/scriggo\n":               {tokenShebangLine},
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
		"<script>s{{a}}</script>{{a}}":                 {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		"<style>s{{a}}</style>{{a}}":                   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style>s{{ render "a" }}t</style>`:            {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<script>s{{ render "a" }}t</script>`:          {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText},
		`<style a="{{b}}"></style>`:                    {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<style a="b">{{1}}</style>`:                   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<script a="{{b}}"></script>`:                  {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<script a="b">{{1}}</script>`:                 {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText},
		`<![CDATA[<script>{{1}}</script>]]>`:           {ast.ContextText},
		`a<![CDATA[<script>{{1}}</script>]]>{{2}}`:     {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<div {{ attr }}>`:                             {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div {{ attr }} {{ attr }}>`:                  {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div{{ attr }}>`:                              {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div {{ attr }}="45">`:                        {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<div "{{ v }}">`:                              {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<a href="">`:                                  {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<A Href="">`:                                  {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<a href=''>`:                                  {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<a href="u">`:                                 {ast.ContextText, ast.ContextQuotedAttr, ast.ContextText, ast.ContextQuotedAttr, ast.ContextText},
		`<a href='u'>`:                                 {ast.ContextText, ast.ContextQuotedAttr, ast.ContextText, ast.ContextQuotedAttr, ast.ContextText},
		`<a href="{{ u }}">`:                           {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<a href={{ u }}>`:                             {ast.ContextText, ast.ContextUnquotedAttr, ast.ContextUnquotedAttr, ast.ContextUnquotedAttr, ast.ContextUnquotedAttr, ast.ContextUnquotedAttr, ast.ContextText},
		`<a href="a{{ p }}">`:                          {ast.ContextText, ast.ContextQuotedAttr, ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<a href="{% if a %}b{% end %}">`:              {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<a {% if a %}{% end %}>`:                      {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<a {% if a %}b{% end %}>`:                     {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<a {% if a %}b=""{% end %}>`:                  {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<a {% if a %}b=''{% end %}>`:                  {ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText, ast.ContextTag, ast.ContextTag, ast.ContextTag, ast.ContextText},
		`<a class="{{ a }}">`:                          {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<a class="c">{{ a }}`:                         {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<a class='c'>{{ a }}`:                         {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<a class=c>{{ a }}`:                           {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<input type="text" disabled class="{{ a }}">`: {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<input type="text" data-value="{{ a }}">`:     {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		"<style>s{{a}}t</style>{{a}}":                  {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style>{{a}}"{{a}}"</style>`:                  {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText},
		`<style>"{{a}}"{{a}}</style>`:                  {ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<style>{{a}}'{{a}}'</style>`:                  {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText},
		`<style>'{{a}}'{{a}}</style>`:                  {ast.ContextText, ast.ContextCSSString, ast.ContextCSSString, ast.ContextCSSString, ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText},
		`<style>'</style>'{{a}}</style>`:               {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<style>"</style>"{{a}}</style>`:               {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		"<script>s{{a}}t</script>{{a}}":                {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script>{{a}}"{{a}}"</script>`:                {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextJSString, ast.ContextJSString, ast.ContextJSString, ast.ContextText},
		`<script>"{{a}}"{{a}}</script>`:                {ast.ContextText, ast.ContextJSString, ast.ContextJSString, ast.ContextJSString, ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText},
		`<script>{{a}}'{{a}}'</script>`:                {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextJSString, ast.ContextJSString, ast.ContextJSString, ast.ContextText},
		`<script>'{{a}}'{{a}}</script>`:                {ast.ContextText, ast.ContextJSString, ast.ContextJSString, ast.ContextJSString, ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText},
		`<script>'</script>'{{a}}</script>`:            {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<script>"</script>"{{a}}</script>`:            {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<script async></script>{{ "a" }}`:             {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},

		`<script type="application/ld+json">s{{a}}t</script>{{a}}`:     {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type="application/ld+json">{{a}}"{{a}}"</script>`:     {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextJSONString, ast.ContextJSONString, ast.ContextJSONString, ast.ContextText},
		`<script type="application/ld+json">"{{a}}"{{a}}</script>`:     {ast.ContextText, ast.ContextJSONString, ast.ContextJSONString, ast.ContextJSONString, ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText},
		`<script type="application/ld+json">{{a}}'{{a}}'</script>`:     {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText},
		`<script type="application/ld+json">'{{a}}'{{a}}</script>`:     {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText},
		`<script type="application/ld+json">'</script>'{{a}}</script>`: {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<script type="application/ld+json">"</script>"{{a}}</script>`: {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
		`<script type="application/ld+json" async></script>{{ "a" }}`:  {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},

		`<script type="application/ld+json">s{{a}}</script>{{a}}`:        {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type="application/ld+json">s{{ render "a" }}t</script>`: {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText},
		`<script type="application/ld+json" a="{{b}}"></script>`:         {ast.ContextText, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextQuotedAttr, ast.ContextText},
		`<script type="application/ld+json" a="b">{{1}}</script>`:        {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText},

		// Script tag with type attribute.
		`<script type="text/javascript">s{{a}}</script>{{a}}`:   {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type=" text/JavaScript ">s{{a}}</script>{{a}}`: {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type=text/javascript>s{{a}}</script>{{a}}`:     {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type= text/javascript >s{{a}}</script>{{a}}`:   {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type="">s{{a}}</script>{{a}}`:                  {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type>s{{a}}</script>{{a}}`:                     {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type="text/plain">s{{a}}</script>{{a}}`:        {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<script type=text/plain>s{{a}}</script>{{a}}`:          {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},

		// Style tag with type attribute.
		`<style type="text/css">s{{a}}t</style>{{a}}`:   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type=" text/CSS ">s{{a}}t</style>{{a}}`: {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type=text/css>s{{a}}t</style>{{a}}`:     {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type="">s{{a}}t</style>{{a}}`:           {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type>s{{a}}t</style>{{a}}`:              {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type="text/plain">s{{a}}t</style>{{a}}`: {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},

		// JavaScript module
		`<style type="module">s{{a}}t</style>{{a}}`:   {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type='module'>s{{a}}t</style>{{a}}`:   {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type=module>s{{a}}t</style>{{a}}`:     {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type="Module">s{{a}}t</style>{{a}}`:   {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
		`<style type=" module ">s{{a}}t</style>{{a}}`: {ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText, ast.ContextHTML, ast.ContextHTML, ast.ContextHTML},
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
	ast.ContextJS: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextJS, ast.ContextJS, ast.ContextJS},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS},
	},
	ast.ContextJSString: {
		`a`:       {ast.ContextText},
		`a{{a}}a`: {ast.ContextText, ast.ContextJSString, ast.ContextJSString, ast.ContextJSString, ast.ContextText},
	},
	ast.ContextJSON: {
		`a`:                             {ast.ContextText},
		`{{a}}`:                         {ast.ContextJSON, ast.ContextJSON, ast.ContextJSON},
		"<script></script>":             {ast.ContextText},
		"<style></style>":               {ast.ContextText},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON},
	},
	ast.ContextJSONString: {
		`a`:       {ast.ContextText},
		`a{{a}}a`: {ast.ContextText, ast.ContextJSONString, ast.ContextJSONString, ast.ContextJSONString, ast.ContextText},
	},
	ast.ContextMarkdown: {
		`a`:                             {ast.ContextText},
		`a{{a}}a`:                       {ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextText},
		"\ta{{a}}\na{{a}}":              {ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		"\ta{{a}}\n\ta{{a}}":            {ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock},
		"    a{{a}}\na{{a}}":            {ast.ContextText, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		"    a{{a}}\n    a{{a}}":        {ast.ContextText, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextText, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock},
		"a\n\t{{a}}":                    {ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		" \t\n\t{{a}}":                  {ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock},
		"\t \n    {{a}}":                {ast.ContextText, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock},
		"{# #}\n\t{{a}}":                {ast.ContextMarkdown, ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		"<style>s{{a}}t</style>{{a}}":   {ast.ContextText, ast.ContextCSS, ast.ContextCSS, ast.ContextCSS, ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		"<script>s{{a}}t</script>{{a}}": {ast.ContextText, ast.ContextJS, ast.ContextJS, ast.ContextJS, ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		`<script type="application/ld+json">s{{a}}t</script>{{a}}`: {ast.ContextText, ast.ContextJSON, ast.ContextJSON, ast.ContextJSON, ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		`a\{{a}\}a`:                {ast.ContextText},
		`a\<a href="{{a}}}">s</a>`: {ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextText},
		"a\\\n{{a}}":               {ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		`a\`:                       {ast.ContextText},
		"a\n\\[\n\t{{a}}":          {ast.ContextText, ast.ContextMarkdown, ast.ContextMarkdown, ast.ContextMarkdown},
		"\t\\{{a}}":                {ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock},
	},
	ast.ContextTabCodeBlock: {
		`a`:       {ast.ContextText},
		`{{a}}`:   {ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock},
		`a{{a}}a`: {ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextText},
		`\{{a}}`:  {ast.ContextText, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock, ast.ContextTabCodeBlock},
	},
	ast.ContextSpacesCodeBlock: {
		`a`:       {ast.ContextText},
		`{{a}}`:   {ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock},
		`a{{a}}a`: {ast.ContextText, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextText},
		`\{{a}}`:  {ast.ContextText, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock, ast.ContextSpacesCodeBlock},
	},
}

var macroAndUsingContextTests = map[string][]ast.Context{
	// Check context for '%}' tokens only.
	"{% macro a %}{% end %}":                                                 {ast.ContextText, ast.ContextText},
	"{% macro a html %}{% end %}":                                            {ast.ContextHTML, ast.ContextText},
	"{% macro a css %}{% end %}":                                             {ast.ContextCSS, ast.ContextText},
	"{% macro a js %}{% end %}":                                              {ast.ContextJS, ast.ContextText},
	"{% macro a json %}{% end %}":                                            {ast.ContextJSON, ast.ContextText},
	"{% macro a markdown %}{% end %}":                                        {ast.ContextMarkdown, ast.ContextText},
	"{% macro a html %}{% macro b %}{% end %}{% end %}":                      {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
	"{% macro a %}{% macro b css %}{% end %}{% end %}":                       {ast.ContextText, ast.ContextCSS, ast.ContextText, ast.ContextText},
	"{% macro a html %}{% macro b css %}{% end %}{% end %}":                  {ast.ContextHTML, ast.ContextCSS, ast.ContextHTML, ast.ContextText},
	"{% macro a html %}{% if true %}{% end %}{% end %}":                      {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
	"{% macro a html %}{% for %}{% end %}{% end %}":                          {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
	"{% macro a html %}{% switch %}{% end %}{% end %}":                       {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
	"{% macro a html %}{% select %}{% end %}{% end %}":                       {ast.ContextHTML, ast.ContextHTML, ast.ContextHTML, ast.ContextText},
	"{% macro a html %}{% var b int %}{% end %}":                             {ast.ContextHTML, ast.ContextHTML, ast.ContextText},
	"{% macro html %}{% end %}":                                              {ast.ContextText, ast.ContextText},
	"{% macro html html %}{% end %}":                                         {ast.ContextHTML, ast.ContextText},
	"{% macro a(b html) %}{% end %}":                                         {ast.ContextText, ast.ContextText},
	"{% macro a(b html) css %}{% end %}":                                     {ast.ContextCSS, ast.ContextText},
	"{% macro /* */ html %}{% end %}":                                        {ast.ContextText, ast.ContextText},
	"{% macro html /* */ %}{% end %}":                                        {ast.ContextText, ast.ContextText},
	"{% macro a js /* */ %}{% end %}":                                        {ast.ContextJS, ast.ContextText},
	"{% macro a\njs\n%}{% end %}":                                            {ast.ContextJS, ast.ContextText},
	"{% macro a\njs\n// comment\n%}{% end %}":                                {ast.ContextJS, ast.ContextText},
	"{% macro a js %}{% for %}{% macro b css %}{% end %}{% end %}{% end %}":  {ast.ContextJS, ast.ContextJS, ast.ContextCSS, ast.ContextJS, ast.ContextJS, ast.ContextText},
	"{% if a %}{% macro b css %}{% end %}{% macro c js %}{% end %}{% end %}": {ast.ContextText, ast.ContextCSS, ast.ContextText, ast.ContextJS, ast.ContextText, ast.ContextText},
	"{% show itea; using %}{% end %}":                                        {ast.ContextText, ast.ContextText},
	"{% show itea; using html %}{% end %}":                                   {ast.ContextHTML, ast.ContextText},
	"{% show itea; using js %}{% end %}":                                     {ast.ContextJS, ast.ContextText},
	"{% show itea; using macro %}{% end %}":                                  {ast.ContextText, ast.ContextText},
	"{% show itea; using macro html %}{% end %}":                             {ast.ContextHTML, ast.ContextText},
	"{% show itea; using /* */ js %}{% end %}":                               {ast.ContextJS, ast.ContextText},
	"{% show itea; using markdown %}{% macro a html %}{% end %}{% end %}":    {ast.ContextMarkdown, ast.ContextHTML, ast.ContextMarkdown, ast.ContextText},
	"{% macro a html %}{% show itea; using js %}{% end %}{% end %}":          {ast.ContextHTML, ast.ContextJS, ast.ContextHTML, ast.ContextText},
}

var positionTests = []struct {
	src string
	pos []ast.Position
}{
	{"a", []ast.Position{
		{1, 1, 0, 0}}},
	{"\n", []ast.Position{
		{1, 1, 0, 0}}},
	{"\n\r", []ast.Position{
		{1, 1, 0, 1}}},
	{"{{a}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 2}, {1, 4, 3, 4}}},
	{"\n{{a}}", []ast.Position{
		{1, 1, 0, 0},
		{2, 1, 1, 2}, {2, 3, 3, 3}, {2, 4, 4, 5}}},
	{"\n\r{{a}}", []ast.Position{
		{1, 1, 0, 1},
		{2, 1, 2, 3}, {2, 3, 4, 4}, {2, 4, 5, 6}}},
	{"{{`a`}}", []ast.Position{
		{1, 1, 0, 1}, {1, 3, 2, 4}, {1, 6, 5, 6}}},
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
	{"5 ", "", 0, 1, 1},
	{" a", "", 0, 1, 1},
	{"a-b ", "a-b", 3, 1, 4},
	{"x:a ", "x:a", 3, 1, 4},
	{"aà ", "aà", 3, 1, 3},
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
	{"x:a=b", "x:a", 0, 4, 1, 5},
	{"data-x=y", "data-x", 0, 7, 1, 8},
}

func testLexerTypes(t *testing.T, test map[string][]tokenTyp, isTemplate bool, format ast.Format) {
TYPES:
	for source, types := range test {
		var lex *lexer
		if isTemplate {
			lex = scanTemplate([]byte(source), format, true, false, true)
		} else {
			lex = scanScript([]byte(source))
		}
		var i int
		for tok := range lex.Tokens() {
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

func TestLexerTypesText(t *testing.T) {
	testLexerTypes(t, typeTestsText, true, ast.FormatText)
}

func TestLexerTypesHTML(t *testing.T) {
	testLexerTypes(t, typeTestsHTML, true, ast.FormatHTML)
}

func TestLexerTypesMarkdown(t *testing.T) {
	testLexerTypes(t, typeTestsMarkdown, true, ast.FormatMarkdown)
}

func TestLexerTypesGo(t *testing.T) {
	testLexerTypes(t, typeTestsGo, false, ast.FormatText)
}

func TestLexerContexts(t *testing.T) {
CONTEXTS:
	for ctx, tests := range contextTests {
		for source, contexts := range tests {
			text := []byte(source)
			lex := &lexer{
				text:           text,
				src:            text,
				line:           1,
				column:         1,
				ctx:            ctx,
				tokens:         make(chan token, 20),
				templateSyntax: true,
				extendedSyntax: true,
			}
			lex.tag.ctx = ast.ContextHTML
			go lex.scan()
			var i int
			for tok := range lex.Tokens() {
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

func TestLexerMacroOrUsingContexts(t *testing.T) {
CONTEXTS:
	for source, contexts := range macroAndUsingContextTests {
		text := []byte(source)
		lex := scanTemplate(text, ast.FormatText, false, false, false)
		var i int
		for tok := range lex.Tokens() {
			if tok.typ == tokenEOF {
				break
			}
			if tok.typ != tokenEndStatement {
				continue
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

func TestPositions(t *testing.T) {
	for _, test := range positionTests {
		var lex = scanTemplate([]byte(test.src), ast.FormatHTML, false, false, false)
		var i int
		for tok := range lex.Tokens() {
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

var endRawIndexTests = []struct {
	src    string
	marker string
	index  int
}{
	{"", "", -1},
	{"a", "", -1},
	{"{", "", -1},
	{"ab {%", "", -1},
	{"ab {% for %} cd", "", -1},
	{"ab {% end %} cd", "", 3},
	{"ab {% end %", "", -1},
	{"ab {% endraw %} cd", "", -1},
	{"ab {% end raw %} cd", "", 3},
	{"ab {% for %} {% end %} cd", "", 13},
	{"ab {% for %} {% end for %} cd", "", -1},
	{"ab {% for %} {% end raw %} cd", "", 13},
	{"ab {%end%} cd", "", 3},
	{"ab {%end raw%} cd", "", 3},
	{"ab {% end {%\t\nend\nraw %}", "", 10},
	{"ab {% end raw", "", -1},
	{"ab {% end raw code %} cd", "code", 3},
	{"ab {% end code %} cd", "code", 3},
	{"ab {% end raw doc %} cd", "code", -1},
	{"ab {% end raw doc %} {% end raw code %} cd", "code", 21},
	{"ab {%\tend\r\n%}", "", 3},
	{"ab {% \x00 end \x12 \x85 %}", "", 3},
	{"ab {% \uFFFD end %}", "", -1},
}

func TestLexRawContent(t *testing.T) {
	for _, test := range endRawIndexTests {
		var marker []byte
		if test.marker != "" {
			marker = []byte(test.marker)
		}
		if i := endRawIndex([]byte(test.src), marker); i != test.index {
			t.Errorf("source: %q, marker: %q: unexpected index %d, expecting %d",
				test.src, test.marker, i, test.index)
		}
	}
}

func TestNoParseShow(t *testing.T) {
	var lex = scanTemplate([]byte("a{{ v }}b"), ast.FormatHTML, false, true, false)
	tokens := lex.Tokens()
	if tok := <-tokens; tok.typ != tokenText {
		t.Errorf("unexpected token %s, expecting text", tok)
	}
	if tok := <-tokens; tok.typ != tokenEOF {
		t.Errorf("unexpected token %s, expecting END", tok)
	}
	lex.Stop()
}

// TestNumbers tests the lexNumber method. The tests are adapted from the
// tests in the "/src/cmd/compile/internal/syntax/scanner_test.go" file in the
// Go repository. That file is copyright "The Go Authors".
func TestNumbers(t *testing.T) {
	for _, test := range []struct {
		typ              tokenTyp
		src, tokens, err string
	}{
		// binaries
		{tokenInt, "0b0", "0b0", ""},
		{tokenInt, "0b1010", "0b1010", ""},
		{tokenInt, "0B1110", "0B1110", ""},

		{tokenInt, "0b", "0b", "binary literal has no digits"},
		{tokenInt, "0b0190", "0b0190", "invalid digit '9' in binary literal"},
		{tokenInt, "0b01a0", "0b01 a0", ""}, // only accept 0-9

		{tokenFloat, "0b.", "0b.", "invalid radix point in binary literal"},
		{tokenFloat, "0b.1", "0b.1", "invalid radix point in binary literal"},
		{tokenFloat, "0b1.0", "0b1.0", "invalid radix point in binary literal"},
		{tokenFloat, "0b1e10", "0b1e10", "'e' exponent requires decimal mantissa"},
		{tokenFloat, "0b1P-1", "0b1P-1", "'P' exponent requires hexadecimal mantissa"},

		{tokenImaginary, "0b10i", "0b10i", ""},
		{tokenImaginary, "0b10.0i", "0b10.0i", "invalid radix point in binary literal"},

		// octals
		{tokenInt, "0o0", "0o0", ""},
		{tokenInt, "0o1234", "0o1234", ""},
		{tokenInt, "0O1234", "0O1234", ""},

		{tokenInt, "0o", "0o", "octal literal has no digits"},
		{tokenInt, "0o8123", "0o8123", "invalid digit '8' in octal literal"},
		{tokenInt, "0o1293", "0o1293", "invalid digit '9' in octal literal"},
		{tokenInt, "0o12a3", "0o12 a3", ""}, // only accept 0-9

		{tokenFloat, "0o.", "0o.", "invalid radix point in octal literal"},
		{tokenFloat, "0o.2", "0o.2", "invalid radix point in octal literal"},
		{tokenFloat, "0o1.2", "0o1.2", "invalid radix point in octal literal"},
		{tokenFloat, "0o1E+2", "0o1E+2", "'E' exponent requires decimal mantissa"},
		{tokenFloat, "0o1p10", "0o1p10", "'p' exponent requires hexadecimal mantissa"},

		{tokenImaginary, "0o10i", "0o10i", ""},
		{tokenImaginary, "0o10e0i", "0o10e0i", "'e' exponent requires decimal mantissa"},

		// 0-octals
		{tokenInt, "0", "0", ""},
		{tokenInt, "0123", "0123", ""},

		{tokenInt, "08123", "08123", "invalid digit '8' in octal literal"},
		{tokenInt, "01293", "01293", "invalid digit '9' in octal literal"},
		{tokenInt, "0F.", "0 F .", ""}, // only accept 0-9
		{tokenInt, "0123F.", "0123 F .", ""},
		{tokenInt, "0123456x", "0123456 x", ""},

		// decimals
		{tokenInt, "1", "1", ""},
		{tokenInt, "1234", "1234", ""},

		{tokenInt, "1f", "1 f", ""}, // only accept 0-9

		{tokenImaginary, "0i", "0i", ""},
		{tokenImaginary, "0678i", "0678i", ""},

		// decimal floats
		{tokenFloat, "0.", "0.", ""},
		{tokenFloat, "123.", "123.", ""},
		{tokenFloat, "0123.", "0123.", ""},

		{tokenFloat, ".0", ".0", ""},
		{tokenFloat, ".123", ".123", ""},
		{tokenFloat, ".0123", ".0123", ""},

		{tokenFloat, "0.0", "0.0", ""},
		{tokenFloat, "123.123", "123.123", ""},
		{tokenFloat, "0123.0123", "0123.0123", ""},

		{tokenFloat, "0e0", "0e0", ""},
		{tokenFloat, "123e+0", "123e+0", ""},
		{tokenFloat, "0123E-1", "0123E-1", ""},

		{tokenFloat, "0.e+1", "0.e+1", ""},
		{tokenFloat, "123.E-10", "123.E-10", ""},
		{tokenFloat, "0123.e123", "0123.e123", ""},

		{tokenFloat, ".0e-1", ".0e-1", ""},
		{tokenFloat, ".123E+10", ".123E+10", ""},
		{tokenFloat, ".0123E123", ".0123E123", ""},

		{tokenFloat, "0.0e1", "0.0e1", ""},
		{tokenFloat, "123.123E-10", "123.123E-10", ""},
		{tokenFloat, "0123.0123e+456", "0123.0123e+456", ""},

		{tokenFloat, "0e", "0e", "exponent has no digits"},
		{tokenFloat, "0E+", "0E+", "exponent has no digits"},
		{tokenFloat, "1e+f", "1e+ f", "exponent has no digits"},
		{tokenFloat, "0p0", "0p0", "'p' exponent requires hexadecimal mantissa"},
		{tokenFloat, "1.0P-1", "1.0P-1", "'P' exponent requires hexadecimal mantissa"},

		{tokenImaginary, "0.i", "0.i", ""},
		{tokenImaginary, ".123i", ".123i", ""},
		{tokenImaginary, "123.123i", "123.123i", ""},
		{tokenImaginary, "123e+0i", "123e+0i", ""},
		{tokenImaginary, "123.E-10i", "123.E-10i", ""},
		{tokenImaginary, ".123E+10i", ".123E+10i", ""},

		// hexadecimals
		{tokenInt, "0x0", "0x0", ""},
		{tokenInt, "0x1234", "0x1234", ""},
		{tokenInt, "0xcafef00d", "0xcafef00d", ""},
		{tokenInt, "0XCAFEF00D", "0XCAFEF00D", ""},

		{tokenInt, "0x", "0x", "hexadecimal literal has no digits"},
		{tokenInt, "0x1g", "0x1 g", ""},

		{tokenImaginary, "0xf00i", "0xf00i", ""},

		// hexadecimal floats
		{tokenFloat, "0x0p0", "0x0p0", ""},
		{tokenFloat, "0x12efp-123", "0x12efp-123", ""},
		{tokenFloat, "0xABCD.p+0", "0xABCD.p+0", ""},
		{tokenFloat, "0x.0189P-0", "0x.0189P-0", ""},
		{tokenFloat, "0x1.ffffp+1023", "0x1.ffffp+1023", ""},

		{tokenFloat, "0x.", "0x.", "hexadecimal literal has no digits"},
		{tokenFloat, "0x0.", "0x0.", "hexadecimal mantissa requires a 'p' exponent"},
		{tokenFloat, "0x.0", "0x.0", "hexadecimal mantissa requires a 'p' exponent"},
		{tokenFloat, "0x1.1", "0x1.1", "hexadecimal mantissa requires a 'p' exponent"},
		{tokenFloat, "0x1.1e0", "0x1.1e0", "hexadecimal mantissa requires a 'p' exponent"},
		{tokenFloat, "0x1.2gp1a", "0x1.2 gp1a", "hexadecimal mantissa requires a 'p' exponent"},
		{tokenFloat, "0x0p", "0x0p", "exponent has no digits"},
		{tokenFloat, "0xeP-", "0xeP-", "exponent has no digits"},
		{tokenFloat, "0x1234PAB", "0x1234P AB", "exponent has no digits"},
		{tokenFloat, "0x1.2p1a", "0x1.2p1 a", ""},

		{tokenImaginary, "0xf00.bap+12i", "0xf00.bap+12i", ""},

		// separators
		{tokenInt, "0b_1000_0001", "0b_1000_0001", ""},
		{tokenInt, "0o_600", "0o_600", ""},
		{tokenInt, "0_466", "0_466", ""},
		{tokenInt, "1_000", "1_000", ""},
		{tokenFloat, "1_000.000_1", "1_000.000_1", ""},
		{tokenImaginary, "10e+1_2_3i", "10e+1_2_3i", ""},
		{tokenInt, "0x_f00d", "0x_f00d", ""},
		{tokenFloat, "0x_f00d.0p1_2", "0x_f00d.0p1_2", ""},

		{tokenInt, "0b__1000", "0b__1000", "'_' must separate successive digits"},
		{tokenInt, "0o60___0", "0o60___0", "'_' must separate successive digits"},
		{tokenInt, "0466_", "0466_", "'_' must separate successive digits"},
		{tokenFloat, "1_.", "1_.", "'_' must separate successive digits"},
		{tokenFloat, "0._1", "0._1", "'_' must separate successive digits"},
		{tokenFloat, "2.7_e0", "2.7_e0", "'_' must separate successive digits"},
		{tokenImaginary, "10e+12_i", "10e+12_i", "'_' must separate successive digits"},
		{tokenInt, "0x___0", "0x___0", "'_' must separate successive digits"},
		{tokenFloat, "0x1.0_p0", "0x1.0_p0", "'_' must separate successive digits"},
	} {
		text := []byte(test.src)
		lex := &lexer{
			text:   text,
			src:    text,
			line:   1,
			column: 1,
			tokens: make(chan token, 1),
		}
		go lex.scan()
		tok, ok := <-lex.Tokens()
		if !ok {
			if lex.err == nil {
				t.Fatal("next called after EOF")
			}
			if test.err == "" {
				t.Fatalf("unexpected error %v, expecting no error", lex.err)
			}
			err, ok := lex.err.(*SyntaxError)
			if !ok {
				t.Fatalf("unexpected error type %T, expecting *SyntaxError", lex.err)
			}
			if err.msg != test.err {
				t.Fatalf("unexpected error %q, expecting error %q", err.msg, test.err)
			}
			continue
		}
		if lex.err != nil {
			t.Fatalf("unexpected error: %q", lex.err)
		}
		if test.err != "" {
			t.Fatalf("unexpected no error, expecting error %q", test.err)
		}
		if tok.typ != test.typ {
			t.Fatalf("unexpected token %s, expecting %s", tok.typ, test.typ)
		}
		for _, expected := range strings.Fields(test.tokens) {
			if tok.typ == tokenEOF {
				t.Fatalf("unexpected EOF, expecting token %q", expected)
			}
			if string(tok.txt) != expected {
				t.Fatalf("unexpected token text %q, expecting %q", tok.txt, expected)
			}
			tok = <-lex.Tokens()
		}
	}
}
