// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"sync"

	"scrigo/internal/compiler/ast"
)

// Cache implements a trees cache used by the parser.
type Cache struct {
	trees map[treeCacheEntry]*ast.Tree
	waits map[treeCacheEntry]*sync.WaitGroup
	sync.Mutex
}

// treeCacheEntry implements a trees cache entry.
type treeCacheEntry struct {
	path string
	ctx  ast.Context
}

// Get returns a tree and true if the tree exists in cache.
//
// If the tree does not exist it returns false and in this
// case a call to Done must be made.
func (c *Cache) Get(path string, ctx ast.Context) (*ast.Tree, bool) {
	entry := treeCacheEntry{path, ctx}
	c.Lock()
	t, ok := c.trees[entry]
	if !ok {
		var wait *sync.WaitGroup
		if wait, ok = c.waits[entry]; ok {
			c.Unlock()
			wait.Wait()
			return c.Get(path, ctx)
		}
		wait = &sync.WaitGroup{}
		wait.Add(1)
		if c.waits == nil {
			c.waits = map[treeCacheEntry]*sync.WaitGroup{entry: wait}
		} else {
			c.waits[entry] = wait
		}
	}
	c.Unlock()
	return t, ok
}

// Add adds a tree to the cache.
//
// Can be called only after a previous call to Get has returned false.
func (c *Cache) Add(path string, ctx ast.Context, tree *ast.Tree) {
	entry := treeCacheEntry{path, ctx}
	c.Lock()
	if c.trees == nil {
		c.trees = map[treeCacheEntry]*ast.Tree{entry: tree}
	} else {
		c.trees[entry] = tree
	}
	c.Unlock()
	return
}

// Done must be called only and only if a previous call to Get has returned
// false.
func (c *Cache) Done(path string, ctx ast.Context) {
	entry := treeCacheEntry{path, ctx}
	c.Lock()
	c.waits[entry].Done()
	delete(c.waits, entry)
	c.Unlock()
	return
}
