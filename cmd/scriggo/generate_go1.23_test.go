//go:build go1.23

//
// Copyright (c) 2024 Open2b Software Snc. All Rights Reserved.
//
// WARNING: This software is protected by international copyright laws.
// Redistribution in part or in whole strictly prohibited.
//

package main

import (
	"fmt"
	"go/importer"
	"testing"
)

func Test_isGeneralInterface(t *testing.T) {
	tests := []struct {
		pkg                string
		declName           string
		isGeneralInterface bool
	}{
		// Interface types.
		{"cmp", "Ordered", true},
		{"fmt", "GoStringer", false},
		{"fmt", "Scanner", false},
		{"fmt", "Stringer", false},

		// Other declaration types (must always be false).
		{"cmp", "Compare", false},
		{"cmp", "Less", false},
		{"os", "ErrInvalid", false},
		{"os", "File", false},
		{"os", "O_RDONLY", false},
	}
	imp := importer.Default()
	for _, test := range tests {
		testName := fmt.Sprintf("%s.%s", test.pkg, test.declName)
		t.Run(testName, func(t *testing.T) {
			pkgs, err := imp.Import(test.pkg)
			if err != nil {
				panic(err)
			}
			decl := pkgs.Scope().Lookup(test.declName)
			if decl == nil {
				t.Fatalf("not found: %q", testName)
			}
			typ := decl.Type()
			got := isGeneralInterface(typ)
			if got != test.isGeneralInterface {
				t.Fatalf("%s: expected isGeneralInterface = %t, got %t", testName, test.isGeneralInterface, got)
			}
		})
	}
}

func Test_isGenericType(t *testing.T) {
	tests := []struct {
		pkg           string
		declName      string
		isGenericType bool
	}{
		// Type types.
		{"cmp", "Ordered", false},
		{"fmt", "GoStringer", false},
		{"fmt", "Scanner", false},
		{"fmt", "Stringer", false},
		{"iter", "Seq", true},
		{"iter", "Seq2", true},

		// Other declaration types (must always be false).
		{"cmp", "Compare", false},
		{"cmp", "Less", false},
		{"os", "ErrInvalid", false},
		{"os", "File", false},
		{"os", "O_RDONLY", false},
	}
	imp := importer.Default()
	for _, test := range tests {
		testName := fmt.Sprintf("%s.%s", test.pkg, test.declName)
		t.Run(testName, func(t *testing.T) {
			pkgs, err := imp.Import(test.pkg)
			if err != nil {
				panic(err)
			}
			decl := pkgs.Scope().Lookup(test.declName)
			if decl == nil {
				t.Fatalf("not found: %q", testName)
			}
			typ := decl.Type()
			got := isGenericType(typ)
			if got != test.isGenericType {
				t.Fatalf("%s: expected isGenericType = %t, got %t", testName, test.isGenericType, got)
			}
		})
	}
}
