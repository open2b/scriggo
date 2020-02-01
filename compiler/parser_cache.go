// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"sync"

	"scriggo/compiler/ast"
)

// cache implements a trees cache used by the parser.
type cache struct {
	trees map[treeCacheEntry]*ast.Tree
	waits map[treeCacheEntry]*sync.WaitGroup
	sync.Mutex
}

// treeCacheEntry implements a trees cache entry.
type treeCacheEntry struct {
	path     string
	language ast.Language
}

// Get returns a tree and true if the tree exists in cache.
//
// If the tree does not exist it returns false and in this
// case a call to Done must be made.
func (c *cache) Get(path string, language ast.Language) (*ast.Tree, bool) {
	entry := treeCacheEntry{path, language}
	c.Lock()
	t, ok := c.trees[entry]
	if !ok {
		var wait *sync.WaitGroup
		if wait, ok = c.waits[entry]; ok {
			c.Unlock()
			wait.Wait()
			return c.Get(path, language)
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
func (c *cache) Add(path string, language ast.Language, tree *ast.Tree) {
	entry := treeCacheEntry{path, language}
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
func (c *cache) Done(path string, language ast.Language) {
	entry := treeCacheEntry{path, language}
	c.Lock()
	c.waits[entry].Done()
	delete(c.waits, entry)
	c.Unlock()
	return
}
