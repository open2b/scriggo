// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type testFormFile struct {
	name    string
	content string
}

var formTests = []struct {
	name   string
	method string
	query  url.Values
	data   url.Values
	multi  map[string]string
	files  map[string]testFormFile
}{
	{"test-get", "GET", url.Values{"a": []string{"b"}}, nil, nil, nil},
	{"test-post", "POST", nil, url.Values{"a": []string{"b"}}, nil, nil},
	{"test-post-with-query", "POST", url.Values{"a": []string{"b"}}, url.Values{"a": []string{"c"}}, nil, nil},
	{"test-multipart", "POST", nil, nil, map[string]string{"a": "b"}, nil},
	{"test-multipart-file", "POST", nil, nil, nil, map[string]testFormFile{
		"f": {"foo.txt", "foo foo"},
	}},
	{"test-multipart-files", "POST", nil, nil, nil, map[string]testFormFile{
		"f1": {"foo1.txt", "foo foo 1"},
		"f2": {"foo2.txt", "foo foo 2"},
	}},
}

func TestForm(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(testFormHandler))
	defer srv.Close()

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, test := range formTests {
		u := srv.URL + "/" + test.name
		if test.query != nil {
			u += "?" + test.query.Encode()
		}
		var res *http.Response
		var err error
		if test.method == "POST" {
			if test.data == nil {
				// multipart/form-data
				var b bytes.Buffer
				w := multipart.NewWriter(&b)
				for k, s := range test.multi {
					fw, _ := w.CreateFormField(k)
					_, _ = io.WriteString(fw, s)
				}
				for k, f := range test.files {
					fw, _ := w.CreateFormFile(k, f.name)
					_, _ = io.WriteString(fw, f.content)
				}
				_ = w.Close()
				res, err = client.Post(srv.URL+"/"+test.name, w.FormDataContentType(), &b)
			} else {
				// application/x-www-form-urlencoded
				res, err = client.PostForm(u, test.data)
			}
		} else {
			res, err = client.Get(u)
		}
		if err != nil {
			t.Fatalf("%s: %s", test.name, err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("%s: unexpected status code %d", test.name, res.StatusCode)
		}
	}

	// Test 400 status code.
	res, err := client.Post(srv.URL+"/test-400", "application/x-www-form-urlencoded", strings.NewReader("a=%zz"))
	if err != nil {
		t.Fatalf("test 400: %s", err)
	}
	if res.StatusCode != 400 {
		t.Fatalf("test 400: unexpected status code %d", res.StatusCode)
	}

	// Test 413 status code.
	const maxFormSize = int(10 << 20)
	res, err = client.Post(srv.URL+"/test-413", "application/x-www-form-urlencoded", strings.NewReader(strings.Repeat(" ", maxFormSize+1)))
	if err != nil {
		t.Fatalf("test 413: %s", err)
	}
	if res.StatusCode != 413 {
		t.Fatalf("test 413: unexpected status code %d", res.StatusCode)
	}

}

func testFormHandler(w http.ResponseWriter, r *http.Request) {

	checkContent := func(f Opener, expected string) error {
		fi, err := f.Open()
		if err != nil {
			return err
		}
		defer fi.Close()
		data, err := io.ReadAll(fi)
		if err != nil {
			return err
		}
		if string(data) != expected {
			return errors.New("unexpected content")
		}
		return nil
	}

	form := NewFormData(r, 1)

	switch r.URL.Path {
	case "/test-get":
		if form.Value("foo") != "" || form.Value("a") != "b" {
			http.Error(w, "", 500)
			return
		}
	case "/test-post":
		if form.Value("foo") != "" || form.Value("a") != "b" {
			http.Error(w, "", 500)
			return
		}
	case "/test-post-with-query":
		values := form.Values()
		if values["foo"] != nil || len(values["a"]) != 2 || values["a"][0] != "c" || values["a"][1] != "b" {
			http.Error(w, "", 500)
			return
		}
	case "/test-multipart":
		form.ParseMultipart()
		if form.Value("foo") != "" || form.Value("a") != "b" {
			http.Error(w, "", 500)
			return
		}
	case "/test-multipart-file":
		form.ParseMultipart()
		foo := form.File("foo")
		f := form.File("f")
		if foo != nil || f.Name() != "foo.txt" || f.Size() != 7 || f.Type() != "application/octet-stream" {
			http.Error(w, "", 500)
			return
		}
		if checkContent(f.(Opener), "foo foo") != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
	case "/test-multipart-files":
		form.ParseMultipart()
		f1 := form.File("f1")
		f2 := form.File("f2")
		if f1.Name() != "foo1.txt" || f1.Size() != 9 || f1.Type() != "application/octet-stream" ||
			f2.Name() != "foo2.txt" || f2.Type() != "application/octet-stream" || f2.Size() != 9 {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		if checkContent(f1.(Opener), "foo foo 1") != nil || checkContent(f2.(Opener), "foo foo 2") != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
	case "/test-400", "/test-413":
		defer func() {
			msg := recover()
			switch msg {
			case ErrBadRequest:
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			case ErrRequestEntityTooLarge:
				http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			panic(msg)
		}()
		_ = form.Value("foo")
	default:
		http.Error(w, "Internal Server Error", 500)
	}

}
