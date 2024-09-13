//
// Copyright (c) 2024 Open2b Software Snc. All Rights Reserved.
//
// WARNING: This software is protected by international copyright laws.
// Redistribution in part or in whole strictly prohibited.
//

package testpkg

// Interface type declarations.

type EmptyInterface interface{}

type Interface1 interface {
	Method()
}
type Interface2 interface {
	struct {
		A int
		B int
	}
	struct {
		A int
		B int
	}
}

type Interface3 interface {
	F1(struct {
		A int
		B int
	})
	F2(struct {
		A int
		B int
	})
}

type GeneralInter1 interface {
	~int
}

type GeneralInter2 interface {
	int
}

type GeneralInter3 interface {
	~int
	String() string
}

type GeneralInter4 interface {
	int | uint
}

type GeneralInter5 interface {
	FunctionType1
}

type GeneralInter6 interface {
	F1(struct {
		A int
		B int
	})
	int | uint
	F2(struct {
		A int
		B int
	})
}

// Type declarations.

type Int1 int

type Int2 Int1

type FunctionType1 func()

type GenericList[E any] []E

type T1 GenericList[int]

type T2[E any] GenericList[E]

// Function declarations.

func F1() {}

func F2[T any](T) {}

type Receiver int

func (Receiver) F3() {}

// Variable declarations.

var V1 int

var V2 GenericList[string]
