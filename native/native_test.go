// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package native

import (
	"testing"
)

func TestCombinedPackage(t *testing.T) {
	pkg1 := DeclarationsPackage{"main", Declarations{"a": 1, "b": 2}}
	pkg2 := DeclarationsPackage{"main2", Declarations{"b": 3, "c": 4, "d": 5}}
	pkg := CombinedPackage{pkg1, pkg2}
	expected := []string{"a", "b", "c", "d"}
	// Test Name.
	if pkg.PackageName() != pkg1.PackageName() {
		t.Fatalf("unexpected name %s, expecting %s", pkg.PackageName(), pkg1.PackageName())
	}
	// Test Lookup.
	for _, name := range expected {
		if decl := pkg.Lookup(name); decl == nil {
			t.Fatalf("unexpected nil, expecting declaration of %s", name)
		}
	}
	if decl := pkg.Lookup("notExistent"); decl != nil {
		t.Fatalf("unexpected %#v for non-existent declaration, expecting nil", decl)
	}
	// Test LookupFunc.
	has := map[string]struct{}{}
	err := pkg.LookupFunc(func(name string, decls Declaration) error {
		if _, ok := has[name]; ok {
			t.Fatalf("unexpected duplicated name %s", name)
		}
		has[name] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, name := range expected {
		if _, ok := has[name]; !ok {
			t.Fatalf("missing name %s from declarations", name)
		}
		delete(has, name)
	}
	if len(has) > 0 {
		var name string
		for name = range has {
			break
		}
		t.Fatalf("unexpected name %s", name)
	}
}
