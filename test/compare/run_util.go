// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"regexp"
	"strings"
)

func splitErrorFromLine(l string) string {
	if index := strings.Index(l, "// ERROR "); index != -1 {
		err := l[index:]
		err = strings.TrimPrefix(err, "// ERROR ")
		err = strings.TrimSpace(err)
		if strings.HasPrefix(err, `"`) && strings.HasSuffix(err, `"`) {
			err = strings.TrimPrefix(err, `"`)
			err = strings.TrimSuffix(err, `"`)
		} else if strings.HasPrefix(err, "`") && strings.HasSuffix(err, "`") {
			err = strings.TrimPrefix(err, "`")
			err = strings.TrimSuffix(err, "`")
			err = regexp.QuoteMeta(err)
		} else {
			panic(fmt.Errorf("expected error must be followed by a string encapsulated in backticks (`) or double quotation marks (\"): %s", err))
		}
		return err
	}
	// If line has a comment containing 'ERROR' is probably a mistake.
	if regexp.MustCompile(`(?m)\/\/\s*?ERROR`).MatchString(l) {
		panic(fmt.Errorf("line '%s' contains string 'ERROR' in a comment but it's not a valid ERROR comment", l))
	}
	return ""
}

// linesWithError returns the list of line numbers of the lines that contains an
// // ERROR comment. Line numbers starts from 1.
func linesWithError(src string) []int {
	lines := []int{}
	for i, line := range strings.Split(src, "\n") {
		if splitErrorFromLine(line) != "" {
			lines = append(lines, i+1)
		}
	}
	return lines
}

type errorcheckTest struct {
	src string
	err string
}

func differentiateSources(src string) []errorcheckTest {
	errLines := linesWithError(src)
	tests := []errorcheckTest{}
	currentError := ""
	currentSrc := strings.Builder{}
	// For every line that contains // ERROR.
	for _, errLine := range errLines {
		currentSrc.Reset()
		currentError = ""
		for i, line := range strings.Split(src, "\n") {
			i := i + 1 // line numbers must start from 1, not 0.
			// If line contains an error but is not the error line, just skip
			// it.
			if splitErrorFromLine(line) != "" && i != errLine {
				continue
			}
			if i == errLine {
				currentError = splitErrorFromLine(line)
			}
			if strings.TrimSpace(line) != "" {
				currentSrc.WriteString(line + "\n")
			}
		}
		tests = append(tests, errorcheckTest{
			src: currentSrc.String(),
			err: currentError,
		})
	}
	return tests
}

// removePrefixFromError removes the prefix of the error (which contains the
// filename and the row and line numbers) if it has one, else it returns the
// error as it is.
func removePrefixFromError(err string) string {
	var re = regexp.MustCompile(`(?m).*?\d+:\d+:\s(.*)`)
	if re.MatchString(err) {
		return re.FindStringSubmatch(err)[1]
	}
	return err
}
