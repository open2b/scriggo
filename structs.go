// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// structFields represents the fields of a struct.
type structFields struct {
	names   []string
	indexOf map[string]int
}

// structs maintains the association between the field names of a struct,
// as they are called in the template, and the field index in the struct.
var structs = struct {
	fields map[reflect.Type]structFields
	sync.RWMutex
}{map[reflect.Type]structFields{}, sync.RWMutex{}}

// getStructFields returns the fields of the struct st.
func getStructFields(st reflect.Value) structFields {
	typ := st.Type()
	structs.RLock()
	fields, ok := structs.fields[typ]
	structs.RUnlock()
	if !ok {
		structs.Lock()
		if fields, ok = structs.fields[typ]; !ok {
			fields = structFields{
				names:   []string{},
				indexOf: map[string]int{},
			}
			n := typ.NumField()
			for i := 0; i < n; i++ {
				fieldType := typ.Field(i)
				if fieldType.PkgPath != "" {
					continue
				}
				name := fieldType.Name
				if tag, ok := fieldType.Tag.Lookup("template"); ok {
					name = parseVarTag(tag)
					if name == "" {
						structs.Unlock()
						panic(fmt.Errorf("template/renderer: invalid tag of field %q", fieldType.Name))
					}
				}
				fields.names = append(fields.names, name)
				fields.indexOf[name] = i
			}
			structs.fields[typ] = fields
		}
		structs.Unlock()
	}
	return fields
}

// parseVarTag parses the tag of a field of a struct that acts as a variable
// and returns the name.
func parseVarTag(tag string) string {
	sp := strings.SplitN(tag, ",", 2)
	if len(sp) == 0 {
		return ""
	}
	name := sp[0]
	if name == "" {
		return ""
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
	}
	return name
}
