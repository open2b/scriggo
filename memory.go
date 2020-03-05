// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scriggo

import (
	"errors"
	"strconv"
	"sync"

	"scriggo/runtime"
)

// SingleMemoryLimiter is a runtime.MemoryLimiter that limits the memory
// available for a single virtual machine execution. It can be used
// concurrently from more goroutines of the same execution. It can not be
// reused after an execution ends.
type SingleMemoryLimiter struct {
	mu   sync.Mutex
	free int // free memory in bytes.
}

// NewSingleMemoryLimiter returns a new SingleMemoryLimiter with a memory
// limited to max bytes. It panics if max is negative.
func NewSingleMemoryLimiter(max int) *SingleMemoryLimiter {
	if max < 0 {
		panic(errors.New("scriggo: NewSingleMemoryLimiter: negative max memory"))
	}
	return &SingleMemoryLimiter{sync.Mutex{}, max}
}

// Reserve reserves bytes of memory. If the memory can not be reserved, it
// returns an error.
func (m *SingleMemoryLimiter) Reserve(env runtime.Env, bytes int) error {
	m.mu.Lock()
	free := m.free
	if free >= 0 {
		free -= bytes
		m.free = free
	}
	m.mu.Unlock()
	if free < 0 {
		return errors.New("cannot allocate " + strconv.Itoa(-m.free) + " bytes")
	}
	return nil
}

// Release releases a previously reserved memory.
func (m *SingleMemoryLimiter) Release(env runtime.Env, bytes int) {
	m.mu.Lock()
	m.free += bytes
	m.mu.Unlock()
}
