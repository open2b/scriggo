// skip

package main

import (
	"strings"
)

func mode(src []byte) string {
	for _, l := range strings.Split(string(src), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if !strings.HasPrefix(l, "//") {
			return ""
		}
		l = strings.TrimPrefix(l, "//")
		l = strings.TrimSpace(l)
		return l
	}
	panic("no directives found")
}
