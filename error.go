//
// author       : Marco Gazerro <gazerro@open2b.com>
// initial date : 26/02/2009
//
// date : 28/01/2013
//
// Copyright (c) 2002-2013 Open2b Software Snc. All Rights Reserved.
//
// WARNING: This software is protected by international copyright laws.
// Redistribution in part or in whole strictly prohibited.
//

package template

import (
	"fmt"
	"html"
	"io/ioutil"
	"strconv"
)

type Error struct {
	Message string
	File    string
	Index   int
	Length  int
}

func (e Error) Error() string {
	return fmt.Sprintf("%v at %v (%v:%v)", e.Message, e.File, e.Index, e.Length)
}

//var newLineReg = regexp.MustCompile(`\r\n|\r|\n`)

func (e Error) Html() string {

	var source = "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Strict//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd\">\n"
	source += "<html xmlns=\"http://www.w3.org/1999/xhtml\">\n<head>\n"
	source += "<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n<title>Error</title>\n"
	source += "<style type=\"text/css\">body { background: #f6f6f6; font-size: 14px; font-family: 'trebuchet ms', arial, verdana, helvetica, sans-serif; padding: 0 1em; }"
	source += "h1 { color: red; font-size: 20px; } h2 { color: #666; font-size: 16px; font-weight: normal; } "
	source += ".container { padding: 1em; border: 1px solid #ccc; background: white; }"
	source += ".rows { background: white; font-family: 'Consolas', 'Monaco', 'Bitstream Vera Sans Mono', 'Courier New', Courier, monospace; border: 1px solid #ddd; margin-bottom: 2em; } "
	source += ".even { background: #f8f8f8; } span.error { color: red; } "
	source += ".lineNumber { background: #eee; color: #333; padding: 0 0.5em; margin-right: 1em; border-right: 1px solid #ccc; }"
	source += ".copyright { margin: 2em 0; text-align: center; }</style>"
	source += "</head>\n<body><div class=\"container\">\n\n"
	source += "<h1>Error: " + html.EscapeString(e.Message) + "</h1>\n"
	source += "<h2>File <em>" + html.EscapeString(e.File) + "</em> at index <em>" + strconv.Itoa(e.Index) + "</em></h2>\n"

	if e.File != "" {
		source += "<div class=\"rows\"><code>\n"
		content, err := ioutil.ReadFile(e.File)
		if err != nil {
			return ""
		}
		source += html.EscapeString(string(content[:e.Index]))
		source += "<span class=\"error\">" + html.EscapeString(string(content[e.Index:e.Index+e.Length])) + "</span>"
		source += html.EscapeString(string(content[e.Index+e.Length:]))
		source += "</code></div>\n"
	}

	source += "</div>\n</body>\n</html>"

	return source
}
