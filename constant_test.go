// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"math/big"
	"testing"
)

var constantNumberTests = []struct {
	num ConstantNumber
	str string
}{
	{newConstantRune('\n'), "'\\n'"},
	{newConstantInt(big.NewInt(5)), "5"},
	{newConstantInt(big.NewInt(2103750168)), "2103750168"},
	{newConstantFloat(big.NewFloat(5.34)), "5.34"},
	{newConstantFloat(big.NewFloat(2103750168.2589183)), "2.1037501682589183e+09"},
}

func TestConstantNumber(t *testing.T) {
	for _, expr := range constantNumberTests {
		str, err := expr.num.String()
		if err != nil {
			t.Errorf("error: %q", err)
			continue
		}
		if str != expr.str {
			t.Errorf("expected: %s, found %s\n", expr.str, str)
			continue
		}
	}
}
