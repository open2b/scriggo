// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import "testing"

var tagValues = []struct {
	value     string
	name      string
	omitempty bool
}{
	{"", "", false},
	{"foo", "foo", false},
	{"foo,boo", "foo", false},
	{",omitempty", "", true},
	{"foo,omitempty", "foo", true},
}

// TestParseTagValue tests the parseTagValue function.
func TestParseTagValue(t *testing.T) {
	for _, v := range tagValues {
		name, omitempty := parseTagValue(v.value)
		if v.name != name {
			t.Fatalf("expecting name %s, got %s", v.name, name)
		}
		if v.omitempty != omitempty {
			t.Fatalf("expecting omitempty %t, got %t", v.omitempty, omitempty)
		}
	}
}
