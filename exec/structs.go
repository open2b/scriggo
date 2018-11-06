//
// Copyright (c) 2016-2018 Open2b Software Snc. All Rights Reserved.
//

package exec

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// fieldNameVersion rappresenta il nome e la versione di un campo di una struct.
type fieldNameVersion struct {
	name    string
	version string
	index   int
}

// structs mantiene l'associazione tra i nomi dei campi di una struct,
// come sono chiamati nel template, e l'indice del campo nella struct.
var structs = struct {
	fields map[reflect.Type][]fieldNameVersion
	sync.RWMutex
}{map[reflect.Type][]fieldNameVersion{}, sync.RWMutex{}}

var errFieldNotExist = errors.New("field does not exist")

// getStructField ritorna il valore del field di nome name della struct st.
// Se il field non esiste ritorna l'errore errFieldNotExist.
func getStructField(st reflect.Value, name, version string) (interface{}, error) {
	for _, field := range getStructFields(st) {
		if field.name == name && (field.version == "" || field.version == version) {
			return st.Field(field.index).Interface(), nil
		}
	}
	return nil, errFieldNotExist
}

// getStructFields ritorna i fields della struct st.
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
						panic(fmt.Errorf("template/exec: invalid tag of field %q", fieldType.Name))
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

// parseVarTag esegue il parsing del tag di un campo di una struct che funge
// da variabile. Ne ritorna il nome e la versione.
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
