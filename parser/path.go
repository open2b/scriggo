//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package parser

import (
	"bytes"
	"fmt"
	"strings"
)

// parsePath ritorna il path letto dal lexer lex.
func parsePath(lex *lexer) (string, error) {
	var tok, ok = <-lex.tokens
	if !ok {
		return "", lex.err
	}
	if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
		return "", fmt.Errorf("unexpected %s, expecting string at %d", tok, tok.pos)
	}
	var path = unquoteString(tok.txt)
	if !isValidFilePath(path) {
		return "", fmt.Errorf("invalid include path %q at %s", path, tok.pos)
	}
	return path, nil
}

// toAbsolutePath unisce dir con path per ottenere un path assoluto.
// dir deve essere assoluto e path relativo. I parametri non vengono
// validati però viene ritornato errore se il path risultante è
// fuori dalla root "/".
func toAbsolutePath(dir, path string) (string, error) {
	if !strings.Contains(path, "..") {
		return dir + path, nil
	}
	var b = []byte(dir + path)
	for i := 0; i < len(b); i++ {
		if b[i] == '/' {
			if b[i+1] == '.' && b[i+2] == '.' {
				if i == 0 {
					return "", fmt.Errorf("template: invalid path %q", path)
				}
				s := bytes.LastIndexByte(b[:i], '/')
				b = append(b[:s+1], b[i+4:]...)
				i = s - 1
			}
		}
	}
	return string(b), nil
}

func isValidDirName(name string) bool {
	// deve essere lungo almeno un carattere e meno di 256
	if name == "" || len(name) >= 256 {
		return false
	}
	// non deve essere '.' e non deve contenere '..'
	if name == "." || strings.Contains(name, "..") {
		return false
	}
	// il primo e ultimo carattere non devono essere spazi
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	// non deve contenere caratteri speciali
	for _, c := range name {
		if ('\x00' <= c && c <= '\x1f') || c == '\x22' || c == '\x2a' || c == '\x2f' ||
			c == '\x3a' || c == '\x3c' || c == '\x3e' || c == '\x3f' || c == '\x5c' ||
			c == '\x7c' || c == '\x7f' {
			return false
		}
	}
	// non deve essere un nome riservato per Window
	name = strings.ToLower(name)
	if name == "con" || name == "prn" || name == "aux" || name == "nul" ||
		(len(name) > 3 && name[0:3] == "com" && '0' <= name[3] && name[3] <= '9') ||
		(len(name) > 3 && name[0:3] == "lpt" && '0' <= name[3] && name[3] <= '9') {
		if len(name) == 4 || name[4] == '.' {
			return false
		}
	}
	return true
}

func isValidFileName(name string) bool {
	// deve essere lungo almeno 3 caratteri e meno di 256
	if len(name) <= 2 || len(name) >= 256 {
		return false
	}
	// il primo e ne l'ultimo carattere non possono essere un punto
	if name[0] == '.' || name[len(name)-1] == '.' {
		return false
	}
	// deve essere presente l'estensione
	var dot = strings.LastIndexByte(name, '.')
	name = strings.ToLower(name)
	var ext = name[dot+1:]
	if strings.IndexByte(ext, '.') >= 0 {
		return false
	}
	// il primo e ultimo carattere non devono essere spazi
	if name[0] == ' ' || name[len(name)-1] == ' ' {
		return false
	}
	// non deve contenere caratteri speciali
	for _, c := range name {
		if ('\x00' <= c && c <= '\x1f') || c == '\x22' || c == '\x2a' || c == '\x2f' ||
			c == '\x3a' || c == '\x3c' || c == '\x3e' || c == '\x3f' || c == '\x5c' ||
			c == '\x7c' || c == '\x7f' {
			return false
		}
	}
	// non deve essere un nome riservato per Window
	if name == "con" || name == "prn" || name == "aux" || name == "nul" ||
		(len(name) > 3 && name[0:3] == "com" && '0' <= name[3] && name[3] <= '9') ||
		(len(name) > 3 && name[0:3] == "lpt" && '0' <= name[3] && name[3] <= '9') {
		if len(name) == 4 || name[4] == '.' {
			return false
		}
	}
	return true
}

// isValidFilePath indica se path è valido come path di un include o extend.
// Sono path validi: '/a', '/a/a', 'a', 'a/a', 'a.a', '../a', 'a/../b'.
// Sono path non validi: '', '/', 'a/', '..', 'a/..'.
func isValidFilePath(path string) bool {
	// deve avere almeno un carattere e non terminare con '/'
	if len(path) < 1 || path[len(path)-1] == '/' {
		return false
	}
	// splitta il path nei vari nomi
	var names = strings.Split(path, "/")
	// i primi nomi devono essere delle directory o '..'
	for i, name := range names[:len(names)-1] {
		// se il primo nome è vuoto...
		if i == 0 && name == "" {
			// ...allora path inizia con '/'
			continue
		}
		if name != ".." && !isValidDirName(name) {
			return false
		}
	}
	// l'ultimo nome deve essere un file
	return isValidFileName(names[len(names)-1])
}
