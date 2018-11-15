//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"testing"
	"time"

	"open2b/template/ast"
)

func TestCache(t *testing.T) {

	path := "/index.html"
	tree := ast.NewTree(path, nil)
	ctx := ast.ContextHTML

	c := cache{}

	// tests one goroutine

	tr, ok := c.get(path, ctx)
	if tr != nil || ok {
		t.Errorf("get not existent tree, unexpected (%v, %t), expecting (nil,false)\n", tr, ok)
	}
	c.done(path, ctx)

	tr, ok = c.get(path, ctx)
	if tr != nil || ok {
		t.Errorf("get not existent tree, unexpected (%v, %t), expecting (nil,false)\n", tr, ok)
	}
	c.add(path, ctx, tree)
	c.done(path, ctx)

	tr, ok = c.get(path, ctx)
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
		c.get(path, ctx)
		go func() {
			steps <- "b: get...\n"
			tr, ok := c.get(path, ctx)
			if tr != nil || ok {
				t.Errorf("get not existent tree, unexpected (%v, %t), expecting (nil,false)\n", tr, ok)
			}
			steps <- "b: get ok\n"
			time.Sleep(10 * time.Millisecond)
			steps <- "b: add...\n"
			c.add(path, ctx, tree)
			steps <- "b: add ok\n"
			time.Sleep(10 * time.Millisecond)
			c.done(path, ctx)
		}()
		time.Sleep(10 * time.Millisecond)
		steps <- "a: done...\n"
		c.done(path, ctx)
		time.Sleep(5 * time.Millisecond)
		steps <- "a: done ok\n"
		steps <- "a: get...\n"
		tr, ok := c.get(path, ctx)
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
