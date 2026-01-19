// Copyright 2026 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"cmp"
	"errors"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/yuin/goldmark/util"
)

type replacement struct {
	start int
	stop  int
	repl  string
}

// linkDestinationReplacer replaces Markdown link destinations with absolute
// URLs. For example, running:
//
//	scriggo build -llms https://example.com src
//
// on the file "src/docs/index.html" transforms [API](api) and
// [API](api.html) into [API](https://example.com/docs/api.md).
//
// URLs with only a query string and/or fragment are left unchanged.
type linkDestinationReplacer struct {
	base *url.URL // base URL passed via the -llms flag (e.g. "https://example.com/")
	dir  string   // directory of the build file, relative to the template root (e.g. "docs")
}

// replace replaces link destinations in Markdown files.
func (r linkDestinationReplacer) replace(dst *bytes.Buffer, src []byte) error {
	if dst == nil {
		return errors.New("dst is nil")
	}
	if src == nil {
		return errors.New("src is nil")
	}
	r.applyReplacements(dst, src, r.collectReplacements(src))
	return nil
}

func (r linkDestinationReplacer) collectReplacements(src []byte) []replacement {

	replacements := make([]replacement, 0)

	var inFence bool
	var fenceChar byte
	var fenceLen int

	for lineStart := 0; lineStart <= len(src); {
		lineEnd := bytes.IndexByte(src[lineStart:], '\n')
		if lineEnd == -1 {
			lineEnd = len(src)
		} else {
			lineEnd += lineStart
		}
		line := src[lineStart:lineEnd]

		if inFence {
			if isFenceClose(line, fenceChar, fenceLen) {
				inFence = false
			}
			lineStart = lineEnd + 1
			continue
		}

		if ok, fc, fl := isFenceStart(line); ok {
			inFence = true
			fenceChar = fc
			fenceLen = fl
			lineStart = lineEnd + 1
			continue
		}

		if isIndentedCode(line) {
			lineStart = lineEnd + 1
			continue
		}

		if start, stop, ok := parseReferenceDefinition(line); ok {
			r.appendReplacement(&replacements, src, lineStart+start, lineStart+stop)
			lineStart = lineEnd + 1
			continue
		}

		r.scanInlineLinks(line, lineStart, src, &replacements)
		lineStart = lineEnd + 1
	}

	return replacements
}

func (r linkDestinationReplacer) applyReplacements(dst *bytes.Buffer, src []byte, replacements []replacement) {
	dst.Reset()
	if len(replacements) == 0 {
		dst.Write(src)
		return
	}
	slices.SortFunc(replacements, func(a, b replacement) int {
		return cmp.Compare(a.start, b.start)
	})
	prev := 0
	for _, r := range replacements {
		if r.start < prev {
			continue
		}
		dst.Write(src[prev:r.start])
		dst.WriteString(r.repl)
		prev = r.stop
	}
	dst.Write(src[prev:])
}

func (r linkDestinationReplacer) appendReplacement(replacements *[]replacement, src []byte, start, stop int) {

	if start < 0 || stop <= start || stop > len(src) {
		return
	}

	destination, err := markdownUnescape(src[start:stop])
	if err != nil {
		return
	}
	u, err := url.Parse(destination)
	if err != nil {
		return
	}

	// Skip absolute URLs and URLs with only query string e/o fragment (e.g. "#index").
	if u.Scheme != "" || (u.Host == "" && u.Path == "") {
		return
	}

	u.Scheme = r.base.Scheme

	// Transform the path only if the URL has no host (e.g. "/info").
	if u.Host == "" {
		u.Host = r.base.Host
		endSlash := strings.HasSuffix(u.Path, "/")
		if path.IsAbs(u.Path) {
			u.Path = path.Join(r.base.Path, u.Path)
		} else {
			u.Path = path.Join(r.base.Path, r.dir, u.Path)
		}
		if endSlash {
			if !strings.HasSuffix(u.Path, "/") {
				u.Path += "/"
			}
		} else {
			ext := path.Ext(u.Path)
			if ext == "" || ext == ".html" {
				if ext == ".html" {
					u.Path = u.Path[:len(u.Path)-len(".html")]
				}
				u.Path += ".md"
			}
		}
	}

	*replacements = append(*replacements, replacement{
		start: start,
		stop:  stop,
		repl:  markdownURLEscape(u.String()),
	})

}

func (r linkDestinationReplacer) scanInlineLinks(line []byte, lineStart int, src []byte, replacements *[]replacement) {
	stack := make([]int, 0, 4)
	codeSpanLen := 0

	for i := 0; i < len(line); {
		c := line[i]

		if codeSpanLen > 0 {
			if c == '`' {
				run := countRun(line, i, '`')
				if run == codeSpanLen && (i+run >= len(line) || line[i+run] != '`') {
					codeSpanLen = 0
					i += run
					continue
				}
			}
			i++
			continue
		}

		if c == '\\' && i+1 < len(line) && util.IsPunct(line[i+1]) {
			i += 2
			continue
		}
		if c == '`' {
			codeSpanLen = countRun(line, i, '`')
			i += codeSpanLen
			continue
		}
		if c == '[' {
			stack = append(stack, i)
			i++
			continue
		}
		if c == ']' {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
				if i+1 < len(line) && line[i+1] == '(' {
					start, stop, end, ok := parseInlineDestination(line, i+2)
					if ok {
						r.appendReplacement(replacements, src, lineStart+start, lineStart+stop)
						i = end
						stack = stack[:0]
						continue
					}
				}
			}
			i++
			continue
		}
		i++
	}
}

func parseInlineDestination(line []byte, pos int) (start, stop, end int, ok bool) {
	pos = skipSpaces(line, pos)
	if pos >= len(line) {
		return 0, 0, 0, false
	}
	if line[pos] == ')' {
		return pos, pos, pos + 1, true
	}
	destStart, destStop, after, ok := parseDestination(line, pos)
	if !ok || destStop <= destStart {
		return 0, 0, 0, false
	}
	end, ok = parseTitleAndClose(line, after)
	if !ok {
		return 0, 0, 0, false
	}
	return destStart, destStop, end, true
}

func parseReferenceDefinition(line []byte) (start, stop int, ok bool) {
	width, pos := util.IndentWidth(line, 0)
	if width > 3 || pos >= len(line) || line[pos] != '[' {
		return 0, 0, false
	}
	labelEnd := findLabelEnd(line, pos+1)
	if labelEnd < 0 {
		return 0, 0, false
	}
	if util.IsBlank(line[pos+1 : labelEnd]) {
		return 0, 0, false
	}
	if labelEnd+1 >= len(line) || line[labelEnd+1] != ':' {
		return 0, 0, false
	}
	i := skipSpaces(line, labelEnd+2)
	destStart, destStop, after, ok := parseDestination(line, i)
	if !ok {
		return 0, 0, false
	}
	after, spaces := skipSpacesCount(line, after)
	if after >= len(line) {
		return destStart, destStop, true
	}
	opener := line[after]
	if opener != '"' && opener != '\'' && opener != '(' {
		if util.IsBlank(line[after:]) {
			return destStart, destStop, true
		}
		return 0, 0, false
	}
	if spaces == 0 {
		return 0, 0, false
	}
	end, ok := parseTitle(line, after)
	if !ok || !util.IsBlank(line[end:]) {
		return 0, 0, false
	}
	return destStart, destStop, true
}

func parseDestination(line []byte, pos int) (start, stop, after int, ok bool) {
	pos = skipSpaces(line, pos)
	if pos >= len(line) {
		return 0, 0, 0, false
	}
	if line[pos] == '<' {
		for i := pos + 1; i < len(line); i++ {
			if line[i] == '\\' && i+1 < len(line) && util.IsPunct(line[i+1]) {
				i++
				continue
			}
			if line[i] == '>' {
				return pos + 1, i, i + 1, true
			}
		}
		return 0, 0, 0, false
	}
	opened := 0
	i := pos
	for i < len(line) {
		c := line[i]
		if c == '\\' && i+1 < len(line) && util.IsPunct(line[i+1]) {
			i += 2
			continue
		}
		if c == '(' {
			opened++
		} else if c == ')' {
			opened--
			if opened < 0 {
				break
			}
		} else if util.IsSpace(c) {
			break
		}
		i++
	}
	if i == pos {
		return 0, 0, 0, false
	}
	return pos, i, i, true
}

func parseTitleAndClose(line []byte, pos int) (end int, ok bool) {
	pos = skipSpaces(line, pos)
	if pos >= len(line) {
		return 0, false
	}
	if line[pos] == ')' {
		return pos + 1, true
	}
	opener := line[pos]
	if opener != '"' && opener != '\'' && opener != '(' {
		return 0, false
	}
	end, ok = parseTitle(line, pos)
	if !ok {
		return 0, false
	}
	end = skipSpaces(line, end)
	if end < len(line) && line[end] == ')' {
		return end + 1, true
	}
	return 0, false
}

func parseTitle(line []byte, pos int) (end int, ok bool) {
	opener := line[pos]
	closer := opener
	if opener == '(' {
		closer = ')'
	}
	for i := pos + 1; i < len(line); i++ {
		c := line[i]
		if c == '\\' && i+1 < len(line) && util.IsPunct(line[i+1]) {
			i++
			continue
		}
		if c == closer {
			return i + 1, true
		}
	}
	return 0, false
}

func findLabelEnd(line []byte, pos int) int {
	for i := pos; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && util.IsPunct(line[i+1]) {
			i++
			continue
		}
		if line[i] == '[' {
			return -1
		}
		if line[i] == ']' {
			return i
		}
	}
	return -1
}

func countRun(line []byte, pos int, c byte) int {
	i := pos
	for i < len(line) && line[i] == c {
		i++
	}
	return i - pos
}

func skipSpaces(line []byte, pos int) int {
	for pos < len(line) && util.IsSpace(line[pos]) {
		pos++
	}
	return pos
}

func skipSpacesCount(line []byte, pos int) (newPos, count int) {
	for pos < len(line) && util.IsSpace(line[pos]) {
		pos++
		count++
	}
	return pos, count
}

func isIndentedCode(line []byte) bool {
	width, _ := util.IndentWidth(line, 0)
	return width >= 4 && !util.IsBlank(line)
}

func isFenceStart(line []byte) (ok bool, fenceChar byte, fenceLen int) {
	width, pos := util.IndentWidth(line, 0)
	if width > 3 || pos >= len(line) {
		return false, 0, 0
	}
	c := line[pos]
	if c != '`' && c != '~' {
		return false, 0, 0
	}
	run := countRun(line, pos, c)
	if run < 3 {
		return false, 0, 0
	}
	if c == '`' && bytes.IndexByte(line[pos+run:], '`') >= 0 {
		return false, 0, 0
	}
	return true, c, run
}

func isFenceClose(line []byte, fenceChar byte, fenceLen int) bool {
	width, pos := util.IndentWidth(line, 0)
	if width > 3 || pos >= len(line) {
		return false
	}
	run := countRun(line, pos, fenceChar)
	if run < fenceLen {
		return false
	}
	return util.IsBlank(line[pos+run:])
}
