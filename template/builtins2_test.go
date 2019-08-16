//+build !darwin

// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

func init() {
	rendererBuiltinTestsInJavaScriptContext = append(rendererBuiltinTestsInJavaScriptContext,
		builtinTest{"var t = {{ t }};", "var t = new Date(\"-012365-03-22T15:19:05.123-07:52\");", Vars{"t": Time(testTime4)}},
		builtinTest{"var t = new Date(\"{{ t }}\");", "var t = new Date(\"-012365-03-22T15:19:05.123-07:52\");", Vars{"t": Time(testTime4)}},
	)
}
