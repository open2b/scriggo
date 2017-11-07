//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"bytes"
	"testing"

	"open2b/template/parser"
)

var execBuiltinTests = []struct {
	src  string
	res  string
	vars map[string]interface{}
}{

	// abbreviate
	{"abbreviate(``,0)", "", nil},
	{"abbreviate(`abc`,0)", "", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,28)", "Lorem ipsum dolor sit amet.", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,27)", "Lorem ipsum dolor sit amet.", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,26)", "Lorem ipsum dolor sit...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,10)", "Lorem...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,8)", "Lorem...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,7)", "...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,6)", "...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,5)", "...", nil},
	{"abbreviate(`Lorem ipsum dolor sit amet.`,4)", "...", nil},
	{"abbreviate(`Lorem ipsum d`,12)", "Lorem...", nil},
	{"abbreviate(`Lorem. Ipsum.`,9)", "Lorem...", nil},
	{"abbreviate(`Lorem, ipsum.`,9)", "Lorem...", nil},

	// abs
	{"abs()", "0", nil},
	{"abs(0)", "0", nil},
	{"abs(1)", "1", nil},
	{"abs(-1)", "1", nil},
	{"abs(3.56)", "3.56", nil},
	{"abs(-3.56)", "3.56", nil},

	// contains
	{"contains()", "true", nil},
	{"contains(``,``)", "true", nil},
	{"contains(`a`,``)", "true", nil},
	{"contains(`abc`,`b`)", "true", nil},
	{"contains(`abc`,`e`)", "false", nil},

	// hasPrefix
	{"hasPrefix()", "true", nil},
	{"hasPrefix(``,``)", "true", nil},
	{"hasPrefix(`a`,``)", "true", nil},
	{"hasPrefix(`abc`,`a`)", "true", nil},
	{"hasPrefix(`abc`,`b`)", "false", nil},

	// hasSuffix
	{"hasSuffix()", "true", nil},
	{"hasSuffix(``,``)", "true", nil},
	{"hasSuffix(`a`,``)", "true", nil},
	{"hasSuffix(`abc`,`c`)", "true", nil},
	{"hasSuffix(`abc`,`b`)", "false", nil},

	// hmac
	{"hmac(`MD5`, ``, ``)", "dOb3KYqcLRaJNfWMAButiA==", nil},
	{"hmac(`MD5`, `hello world!`, ``)", "POUE2/xvWDT8UjcXJ4d/hQ==", nil},
	{"hmac(`MD5`, ``, `secret`)", "XI2wPwTOwPQ7ywYAI5FBkA==", nil},
	{"hmac(`MD5`, `hello world!`, `secret`)", "CgRh4Q6JUG18MaFFZjvtkw==", nil},
	{"hmac(`SHA-1`, ``, ``)", "+9sdGxiqbAgyS31ktx+3Y3BpDh0=", nil},
	{"hmac(`SHA-1`, `hello world!`, ``)", "Cs2Lo6MmqAmr0Qj3JXmz/wJnhDg=", nil},
	{"hmac(`SHA-1`, ``, `secret`)", "Ja9hdKD87MTTRmgKcrfOZEuaiOg=", nil},
	{"hmac(`SHA-1`, `hello world!`, `secret`)", "pN9fnSN6sMoyQfBCvPYFmk70kcQ=", nil},
	{"hmac(`SHA-256`, ``, ``)", "thNnmggU2ex3L5XXeMNfxf8Wl8STcVZTxscSFEKSxa0=", nil},
	{"hmac(`SHA-256`, `hello world!`, ``)", "7/WCWbmktkh3Gig/DI7JWORlJ0gUpKhebIYJG4iMxJw=", nil},
	{"hmac(`SHA-256`, ``, `secret`)", "+eZuF5tnR65UEI+C+K3os8Jddv0wr95sOVgixTAZYWk=", nil},
	{"hmac(`SHA-256`, `hello world!`, `secret`)", "cgaXMb8pG0Y67LIYvCJ6vOPUA9dtpn+u8tSNPLQ7L1Q=", nil},

	// html
	{"html(``)", "", nil},
	{"html(`a`)", "a", nil},
	{"html(`<a>`)", "<a>", nil},
	{"html(a)", "<a>", map[string]interface{}{"a": "<a>"}},
	{"html(a)", "<a>", map[string]interface{}{"a": HTML("<a>")}},

	// index
	{"index(``,``)", "0", nil},
	{"index(`a`,``)", "0", nil},
	{"index(`ab€c`,`a`)", "0", nil},
	{"index(`ab€c`,`b`)", "1", nil},
	{"index(`ab€c`,`€`)", "2", nil},
	{"index(`ab€c`,`c`)", "3", nil},
	{"index(`ab€c`,`d`)", "-1", nil},
	{"index(`ab€c`,`ab`)", "0", nil},
	{"index(`ab€c`,`b€`)", "1", nil},
	{"index(`ab€c`,`bc`)", "-1", nil},

	// indexAny
	{"indexAny(``,``)", "-1", nil},
	{"indexAny(`a`,``)", "-1", nil},
	{"indexAny(`ab€c`,`a`)", "0", nil},
	{"indexAny(`ab€c`,`b`)", "1", nil},
	{"indexAny(`ab€c`,`€`)", "2", nil},
	{"indexAny(`ab€c`,`c`)", "3", nil},
	{"index(`ab€c`,`d`)", "-1", nil},
	{"indexAny(`ab€c`,`ab`)", "0", nil},
	{"indexAny(`ab€c`,`ac`)", "0", nil},
	{"indexAny(`ab€c`,`cb`)", "1", nil},
	{"indexAny(`ab€c`,`c€`)", "2", nil},
	{"indexAny(`ab€c`,`ef`)", "-1", nil},

	// int
	{"int(0)", "0", nil},
	{"int(1)", "1", nil},
	{"int(0.5)", "0", nil},
	{"int(-0.5)", "0", nil},
	{"int(3.56)", "3", nil},

	// join
	{"join(a, ``)", "", map[string]interface{}{"a": []string(nil)}},
	{"join(a, ``)", "", map[string]interface{}{"a": []string{}}},
	{"join(a, ``)", "a", map[string]interface{}{"a": []string{"a"}}},
	{"join(a, ``)", "ab", map[string]interface{}{"a": []string{"a", "b"}}},
	{"join(a, `,`)", "a,b,c", map[string]interface{}{"a": []string{"a", "b", "c"}}},

	// lastIndex
	{"lastIndex(``,``)", "0", nil},
	{"lastIndex(`a`,``)", "1", nil},
	{"lastIndex(``,`a`)", "-1", nil},
	{"lastIndex(`ab€ac€`,`a`)", "3", nil},
	{"lastIndex(`ab€ac€`,`b`)", "1", nil},
	{"lastIndex(`ab€ac€`,`€`)", "5", nil},
	{"lastIndex(`ab€ac€`,`c`)", "4", nil},
	{"lastIndex(`ab€ac€`,`d`)", "-1", nil},
	{"lastIndex(`ab€ac€`,`ab`)", "0", nil},
	{"lastIndex(`ab€acb€`,`b€`)", "5", nil},
	{"lastIndex(`ab€acb€`,`bc`)", "-1", nil},

	// max
	{"max(0, 0)", "0", nil},
	{"max(5, 0)", "5", nil},
	{"max(0, 7)", "7", nil},
	{"max(5, 7)", "7", nil},
	{"max(7, 5)", "7", nil},
	{"max(-7, 5)", "5", nil},
	{"max(7, -5)", "7", nil},
	{"max(5.5, 7.5)", "7.5", nil},
	{"max(7.5, 5.5)", "7.5", nil},
	{"max(-7.5, 5.5)", "5.5", nil},
	{"max(7.5, -5.5)", "7.5", nil},
	{"max(0.0000000000000000000000000000001, 0.0000000000000000000000000000002)", "0.0000000000000000000000000000002", nil},

	// md5
	{"md5(``)", "d41d8cd98f00b204e9800998ecf8427e", nil},
	{"md5(`hello world!`)", "fc3ff98e8c6a0d3087d515c0473f8677", nil},

	// min
	{"min(0, 0)", "0", nil},
	{"min(5, 0)", "0", nil},
	{"min(0, 7)", "0", nil},
	{"min(5, 7)", "5", nil},
	{"min(7, 5)", "5", nil},
	{"min(-7, 5)", "-7", nil},
	{"min(7, -5)", "-5", nil},
	{"min(5.5, 7.5)", "5.5", nil},
	{"min(7.5, 5.5)", "5.5", nil},
	{"min(-7.5, 5.5)", "-7.5", nil},
	{"min(7.5, -5.5)", "-5.5", nil},
	{"min(0.0000000000000000000000000000001, 0.0000000000000000000000000000002)", "0.0000000000000000000000000000001", nil},

	// repeat
	{"repeat(``, 0)", "", nil},
	{"repeat(`a`, 0)", "", nil},
	{"repeat(`a`, 1)", "a", nil},
	{"repeat(`a`, 5)", "aaaaa", nil},
	{"repeat(`€`, 3)", "€€€", nil},
	{"repeat(`€€`, 3)", "€€€€€€", nil},

	// replace
	{"replace()", "", nil},
	{"replace(``)", "", nil},
	{"replace(``, ``)", "", nil},
	{"replace(``, ``, ``)", "", nil},
	{"replace(`abc`, `b`, `e`)", "aec", nil},
	{"replace(`abc`, `b`, `€`)", "a€c", nil},
	{"replace(`abcbcba`, `b`, `e`)", "aececea", nil},

	// round
	{"round()", "0", nil},
	{"round(0)", "0", nil},
	{"round(5.3752, 2)", "5.38", nil},

	// sha1
	{"sha1(``)", "da39a3ee5e6b4b0d3255bfef95601890afd80709", nil},
	{"sha1(`hello world!`)", "430ce34d020724ed75a196dfc2ad67c77772d169", nil},

	// sha256
	{"sha256(``)", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", nil},
	{"sha256(`hello world!`)", "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9", nil},

	// split
	{"split()", "", nil},
	{"split(``, ``)", "", nil},
	{"split(`a`, ``)", "a", nil},
	{"split(`ab`, ``)", "a, b", nil},
	{"split(`a,b,c`, `,`)", "a, b, c", nil},
	{"split(`a,b,c,`, `,`)", "a, b, c, ", nil},

	// splitAfter
	{"splitAfter()", "", nil},
	{"splitAfter(``, ``)", "", nil},
	{"splitAfter(`a`, ``)", "a", nil},
	{"splitAfter(`ab`, ``)", "a, b", nil},
	{"splitAfter(`a,b,c`, `,`)", "a,, b,, c", nil},
	{"splitAfter(`a,b,c,`, `,`)", "a,, b,, c,, ", nil},

	// title
	{"title()", "", nil},
	{"title(``)", "", nil},
	{"title(`a`)", "A", nil},
	{"title(`5`)", "5", nil},
	{"title(`€`)", "€", nil},
	{"title(`ab`)", "Ab", nil},
	{"title(`5a`)", "5a", nil},
	{"title(`ab cd`)", "Ab Cd", nil},

	// toLower
	{"toLower()", "", nil},
	{"toLower(``)", "", nil},
	{"toLower(`a`)", "a", nil},
	{"toLower(`A`)", "a", nil},
	{"toLower(`aB`)", "ab", nil},
	{"toLower(`aBCd`)", "abcd", nil},
	{"toLower(`èÈ`)", "èè", nil},

	// toTitle
	{"toTitle()", "", nil},
	{"toTitle(``)", "", nil},
	{"toTitle(`a`)", "A", nil},
	{"toTitle(`5`)", "5", nil},
	{"toTitle(`€`)", "€", nil},
	{"toTitle(`ab`)", "AB", nil},
	{"toTitle(`5a`)", "5A", nil},
	{"toTitle(`ab cd`)", "AB CD", nil},

	// toUpper
	{"toUpper()", "", nil},
	{"toUpper(``)", "", nil},
	{"toUpper(`A`)", "A", nil},
	{"toUpper(`a`)", "A", nil},
	{"toUpper(`Ab`)", "AB", nil},
	{"toUpper(`AbcD`)", "ABCD", nil},
	{"toUpper(`Èè`)", "ÈÈ", nil},

	// trim
	{"trim()", "", nil},
	{"trim(``)", "", nil},
	{"trim(` `)", "", nil},
	{"trim(` a`)", "a", nil},
	{"trim(`a `)", "a", nil},
	{"trim(` a `)", "a", nil},
	{"trim(` a b  `)", "a b", nil},
}

var execRandomBuiltinTests = []struct {
	src  string
	seed int64
	res  string
	vars map[string]interface{}
}{
	// rand
	{"rand(0)", 1, "5577006791947779410", nil},
	{"rand(0)", 2, "1543039099823358511", nil},
	{"rand(1)", 1, "0", nil},
	{"rand(1)", 2, "0", nil},
	{"rand(2)", 1, "1", nil},
	{"rand(2)", 2, "0", nil},
	{"rand(100)", 1, "81", nil},
	{"rand(100)", 2, "86", nil},
	{"rand(100)", 3, "8", nil},
	{"rand(100)", 4, "29", nil},

	// shuffle
	{"shuffle(s)", 1, "", map[string]interface{}{"s": []int{}}},
	{"shuffle(s)", 1, "1", map[string]interface{}{"s": []int{1}}},
	{"shuffle(s)", 1, "1, 2", map[string]interface{}{"s": []int{1, 2}}},
	{"shuffle(s)", 2, "2, 1", map[string]interface{}{"s": []int{1, 2}}},
	{"shuffle(s)", 1, "1, 2, 3", map[string]interface{}{"s": []int{1, 2, 3}}},
	{"shuffle(s)", 2, "3, 1, 2", map[string]interface{}{"s": []int{1, 2, 3}}},
	{"shuffle(s)", 3, "1, 3, 2", map[string]interface{}{"s": []int{1, 2, 3}}},
	{"shuffle(s)", 1, "a, b, c", map[string]interface{}{"s": []string{"a", "b", "c"}}},
	{"shuffle(s)", 2, "c, a, b", map[string]interface{}{"s": []string{"a", "b", "c"}}},
	{"shuffle(s)", 3, "a, c, b", map[string]interface{}{"s": []string{"a", "b", "c"}}},
}

func TestExecBuiltin(t *testing.T) {
	for _, expr := range execBuiltinTests {
		var tree, err = parser.Parse([]byte("{{" + expr.src + "}}"))
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, "")
		err = env.Execute(b, expr.vars)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
		}
	}
}

func TestExecRandomBuiltin(t *testing.T) {
	for _, expr := range execRandomBuiltinTests {
		var tree, err = parser.Parse([]byte("{{" + expr.src + "}}"))
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var b = &bytes.Buffer{}
		var env = NewEnv(tree, "")
		testSeed = expr.seed
		err = env.Execute(b, expr.vars)
		if err != nil {
			t.Errorf("source: %q, %s\n", expr.src, err)
			continue
		}
		var res = b.String()
		if res != expr.res {
			t.Errorf("source: %q, unexpected %q, expecting %q\n", expr.src, res, expr.res)
		}
	}
}
