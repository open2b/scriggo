// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"testing"
	"time"

	"scrigo/internal/compiler/ast"
)

func TestCache(t *testing.T) {

	path := "/index.html"
	tree := ast.NewTree(path, nil, ast.ContextHTML)
	ctx := ast.ContextHTML

	c := cache{}

	// tests one goroutine

	tr, ok := c.Get(path, ctx)
	if tr != nil || ok {
		t.Errorf("get not existent tree, unexpected (%v, %t), expecting (nil,false)\n", tr, ok)
	}
	c.Done(path, ctx)

	tr, ok = c.Get(path, ctx)
	if tr != nil || ok {
		t.Errorf("get not existent tree, unexpected (%v, %t), expecting (nil,false)\n", tr, ok)
	}
	c.Add(path, ctx, tree)
	c.Done(path, ctx)

	tr, ok = c.Get(path, ctx)
	if tr != tree || !ok {
		t.Errorf("get tree, unexpected (%v, %t), expecting (%v,true)\n", tr, ok, tree)
	}

	// tests more goroutine

	const expectedSteps = `
b: get...
a: done...
b: get ok
a: done ok
a: get...
b: add...
b: add ok
a: get ok
`

	c = cache{}
	steps := make(chan string, 10)
	go func() {
		c.Get(path, ctx)
		go func() {
			steps <- "b: get...\n"
			tr, ok := c.Get(path, ctx)
			if tr != nil || ok {
				t.Errorf("get not existent tree, unexpected (%v, %t), expecting (nil,false)\n", tr, ok)
			}
			steps <- "b: get ok\n"
			time.Sleep(10 * time.Millisecond)
			steps <- "b: add...\n"
			c.Add(path, ctx, tree)
			steps <- "b: add ok\n"
			time.Sleep(10 * time.Millisecond)
			c.Done(path, ctx)
		}()
		time.Sleep(10 * time.Millisecond)
		steps <- "a: done...\n"
		c.Done(path, ctx)
		time.Sleep(5 * time.Millisecond)
		steps <- "a: done ok\n"
		steps <- "a: get...\n"
		tr, ok := c.Get(path, ctx)
		if tr != tree || !ok {
			t.Errorf("unexpected (%v, %t), expecting (%v,true)\n", tr, ok, tree)
		}
		steps <- "a: get ok\n"
		close(steps)
	}()

	executedSteps := "\n"
	for step := range steps {
		executedSteps += step
	}
	if executedSteps != expectedSteps {
		t.Errorf("unexpected steps: %s, expecting steps: %s\n", executedSteps, expectedSteps)
	}

}
