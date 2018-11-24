// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package renderer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// fieldNameVersion represents the name and version of a field in a struct.
type fieldNameVersion struct {
	name    string
	version string
	index   int
}

// structs maintains the association between the field names of a struct,
// as they are called in the template, and the field index in the struct.
var structs = struct {
	fields map[reflect.Type][]fieldNameVersion
	sync.RWMutex
}{map[reflect.Type][]fieldNameVersion{}, sync.RWMutex{}}

var errFieldNotExist = errors.New("field does not exist")

// getStructField returns the value of the field named name of the struct st.
// If the field does not exist, the errFieldNotExist error is returned.
func getStructField(st reflect.Value, name, version string) (interface{}, error) {
	for _, field := range getStructFields(st) {
		if field.name == name && (field.version == "" || field.version == version) {
			return st.Field(field.index).Interface(), nil
		}
	}
	return nil, errFieldNotExist
}

// getStructFields returns the fields of the struct st.
func getStructFields(st reflect.Value) []fieldNameVersion {
	typ := st.Type()
	structs.RLock()
	fields, ok := structs.fields[typ]
	structs.RUnlock()
	if !ok {
		structs.Lock()
		if fields, ok = structs.fields[typ]; !ok {
			n := typ.NumField()
			fields = make([]fieldNameVersion, 0, n)
			for i := 0; i < n; i++ {
				fieldType := typ.Field(i)
				if fieldType.PkgPath != "" {
					continue
				}
				if tag, ok := fieldType.Tag.Lookup("template"); ok {
					var field fieldNameVersion
					field.name, field.version = parseVarTag(tag)
					if field.name == "" {
						structs.Unlock()
						panic(fmt.Errorf("template/renderer: invalid tag of field %q", fieldType.Name))
					}
					field.index = i
					fields = append(fields, field)
				} else {
					fields = append(fields, fieldNameVersion{fieldType.Name, "", i})
				}
			}
			structs.fields[typ] = fields
		}
		structs.Unlock()
	}
	return fields
}

// parseVarTag parses the tag of a field of a struct that acts as a variable.
// It returns the name and version.
func parseVarTag(tag string) (string, string) {
	sp := strings.SplitN(tag, ",", 2)
	if len(sp) == 0 {
		return "", ""
	}
	name := sp[0]
	if name == "" {
		return "", ""
	}
	for _, r := range name {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", ""
		}
	}
	var version string
	if len(sp) == 2 {
		version = sp[1]
		if version == "" {
			return "", ""
		}
	}
	return name, version
}
