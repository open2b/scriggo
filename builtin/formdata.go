// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

const maxInt = int64(^uint(0) >> 1)

// ErrBadRequest is the error that occurs when parsing a malformed HTTP
// request body or query string in [FormData] methods.
var ErrBadRequest = errors.New("form: bad request")

// ErrRequestEntityTooLarge is the error that occurs when the HTTP
// request's body is too large
var ErrRequestEntityTooLarge = errors.New("form: request entity too large")

// File represents a file. Files that can be opened also implement [Opener].
type File interface {
	Name() string // name.
	Type() string // type, as mime type.
	Size() int    // size, in bytes.
}

// Opener is implemented by a [File] that can be opened. [Opener] is not intended
// to be used as a builtin type but it can be used by builtins to open a file.
type Opener interface {
	// Open opens the file.
	Open() (io.ReadSeekCloser, error)
}

// formFile implements a File returned by the File and Files methods of the
// FormData type. It is not intended to be used as a builtin type.
type formFile struct {
	fh *multipart.FileHeader
}

// Name returns the name of the file.
func (f formFile) Name() string {
	return f.fh.Filename
}

// Type returns the type of the file.
func (f formFile) Type() string {
	return f.fh.Header.Get("Content-Type")
}

// Size returns the size of the file in bytes.
func (f formFile) Size() int {
	return int(f.fh.Size)
}

// Open opens the file.
func (f formFile) Open() (io.ReadSeekCloser, error) {
	return f.fh.Open()
}

// FormData contains the form data from the query string and body of an HTTP
// request. Values and files passed as multipart/form-data are only available
// after ParseMultipart method is called.
type FormData struct {
	request   *http.Request
	data      *formData
	maxMemory int64
}

// formData contains the data of the FormData type.
type formData struct {
	values url.Values
	files  map[string][]File
}

// NewFormData returns a new [FormData] value. maxMemory is passed as is to the
// r.ParseMultipartForm method.
func NewFormData(r *http.Request, maxMemory int64) FormData {
	return FormData{request: r, data: &formData{}, maxMemory: maxMemory}
}

// ParseMultipart parses a request body as multipart/form-data. If the body
// is not multipart/form-data, it does nothing. It should be called before
// calling the [FormData.File] and [FormData.Files] methods and can be called
// multiple times.
//
// It panics with [ErrBadRequest] if the request is not valid,
// [ErrRequestEntityTooLarge] if the length of the body is too large or another
// error if another error occurs.
func (form FormData) ParseMultipart() {
	err := form.request.ParseMultipartForm(form.maxMemory)
	if err == http.ErrNotMultipart {
		return
	}
	if err != nil {
		if isMultipartFormError(err) {
			panic(ErrBadRequest)
		} else if err.Error() == "http: POST too large" {
			panic(ErrRequestEntityTooLarge)
		} else if ct := form.request.Header.Get("Content-Type"); ct != "" {
			if _, _, e := mime.ParseMediaType(ct); e != nil {
				panic(ErrBadRequest)
			}
		}
		panic("form: " + err.Error())
	}
	form.data.values = form.request.MultipartForm.Value
	form.data.files = make(map[string][]File, len(form.request.MultipartForm.File))
	for field, fhs := range form.request.MultipartForm.File {
		files := make([]File, 0, len(fhs))
		for _, fh := range fhs {
			if fh.Size <= maxInt {
				files = append(files, formFile{fh: fh})
			}
		}
		if len(files) > 0 {
			form.data.files[field] = files
		}
	}
}

// parse parses the body and query string of the request. It is called when
// the Value and Values methods are called.
//
// It panics with [ErrBadRequest] if the request is not valid,
// [ErrRequestEntityTooLarge] if the length of the body is too large or another
// error if another error occurs.
func (form FormData) parse() {
	err := form.request.ParseForm()
	if err != nil {
		if _, ok := err.(url.EscapeError); ok {
			panic(ErrBadRequest)
		} else if s := err.Error(); s == "invalid semicolon separator in query" {
			panic(ErrBadRequest)
		} else if s == "http: POST too large" {
			panic(ErrRequestEntityTooLarge)
		} else if ct := form.request.Header.Get("Content-Type"); ct != "" {
			if _, _, e := mime.ParseMediaType(ct); e != nil {
				panic(ErrBadRequest)
			}
		}
		panic("form:" + err.Error())
	}
	form.data.values = form.request.Form
}

// Value returns the first form data value associated with the given field. If
// there are no files associated with the field, it returns an empty string.
//
// It panics with [ErrBadRequest] if the request is not valid,
// [ErrRequestEntityTooLarge] if the length of the body is too large or another
// error if another error occurs.
func (form FormData) Value(field string) string {
	if form.data.values == nil {
		form.parse()
	}
	return form.data.values.Get(field)
}

// Values returns the parsed form data, including both the URL field's query
// parameters and the POST form data.
//
// It panics with [ErrBadRequest] if the request is not valid,
// [ErrRequestEntityTooLarge] if the length of the body is too large or another
// error if another error occurs.
func (form FormData) Values() map[string][]string {
	if form.data.values == nil {
		form.parse()
	}
	return form.data.values
}

// File returns the first file associated with the given field. It returns nil
// if there are no values associated with the field.
//
// Call File only after [FormData.ParseMultipart] is called.
func (form FormData) File(field string) File {
	if form.data.files == nil {
		return nil
	}
	files := form.data.files[field]
	if len(files) == 0 {
		return nil
	}
	return files[0]
}

// Files returns the parsed files of a multipart form. It returns a non nil
// map, if ParseMultipart has been called.
//
// Call Files only after [FormData.ParseMultipart] is called.
func (form FormData) Files() map[string][]File {
	return form.data.files
}

// isMultipartFormError reports whether err is an error due to an invalid
// multipart message returned from the (*http.Request).ParseMultipartForm
// method.
func isMultipartFormError(err error) bool {
	if err == http.ErrMissingBoundary {
		return true
	}
	if _, ok := err.(url.EscapeError); ok {
		return true
	}
	s := err.Error()
	if s == "invalid semicolon separator in query" || s == "multipart: boundary is empty" {
		return true
	}
	if strings.HasPrefix(s, "multipart: NextPart: ") {
		return true
	}
	if strings.HasPrefix(s, "multipart: unexpected line in Next(): ") {
		return true
	}
	if err, ok := err.(textproto.ProtocolError); ok {
		s = err.Error()
		if strings.HasPrefix(s, "malformed MIME header initial line: ") {
			return true
		}
		if strings.HasPrefix(s, "malformed MIME header line: ") {
			return true
		}
	}
	return false
}
