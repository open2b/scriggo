// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// decimalKey implements a key of type decimal.
type decimalKey string

// Map implements the map values.
type Map map[interface{}]interface{}

// Delete deletes the value for a key.
func (m Map) Delete(key interface{}) {
	if k, valid := mapKeyImp(key); valid {
		delete(m, k)
	}
}

// Len returns the length of the map.
func (m Map) Len() int {
	return len(m)
}

// Load returns the value stored in the map for a key, or nil if no value is
// present. The ok result indicates whether value was found in the map.
func (m Map) Load(key interface{}) (value interface{}, ok bool) {
	if k, valid := mapKeyImp(key); valid {
		value, ok = m[k]
	}
	return
}

// Range calls f sequentially for each key and value present in the map. If f
// returns false, range stops the iteration.
func (m Map) Range(f func(key, value interface{}) bool) {
	for k, v := range m {
		if dk, ok := k.(decimalKey); ok {
			k, _ = decimal.NewFromString(string(dk))
		}
		if !f(k, v) {
			return
		}
	}
}

// Store sets the value for a key.
func (m Map) Store(key, value interface{}) {
	if m == nil {
		panic("assignment to entry in nil map")
	}
	if k, valid := mapKeyImp(key); valid {
		m[k] = value
		return
	}
	panic(fmt.Sprintf("hash of unhashable type %T", key))
}

func mapKeyImp(key interface{}) (interface{}, bool) {
	switch d := key.(type) {
	case nil, string, HTML, int, bool:
		return key, true
	case decimal.Decimal:
		if !d.LessThan(minIntAsDecimal) && !maxIntAsDecimal.LessThan(d) {
			p := d.IntPart()
			if decimal.New(p, 0).Equal(d) {
				return int(p), true
			}
		}
		return decimalKey(d.String()), true
	}
	return nil, false
}
