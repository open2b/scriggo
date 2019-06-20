package main

import "testing"

func BenchmarkGeneratePackage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, v, err := goPackageToDeclarations("fmt", "")
		if err != nil {
			b.Fatal(err)
		}
		_ = v
	}
}
