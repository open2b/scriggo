// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"time"

	"github.com/open2b/scriggo/builtin"
	"github.com/open2b/scriggo/native"
)

var globals = native.Declarations{
	// crypto
	"hmacSHA1":   builtin.HmacSHA1,
	"hmacSHA256": builtin.HmacSHA256,
	"sha1":       builtin.Sha1,
	"sha256":     builtin.Sha256,

	// debug
	"version": version(),

	// encoding
	"base64":            builtin.Base64,
	"hex":               builtin.Hex,
	"marshalJSON":       builtin.MarshalJSON,
	"marshalJSONIndent": builtin.MarshalJSONIndent,
	"marshalYAML":       builtin.MarshalYAML,
	"md5":               builtin.Md5,
	"unmarshalJSON":     builtin.UnmarshalJSON,
	"unmarshalYAML":     builtin.UnmarshalYAML,

	// html
	"htmlEscape": builtin.HtmlEscape,

	// math
	"abs": builtin.Abs,
	"max": builtin.Max,
	"min": builtin.Min,
	"pow": builtin.Pow,

	// net
	"File":        reflect.TypeOf((*builtin.File)(nil)).Elem(),
	"FormData":    reflect.TypeOf(builtin.FormData{}),
	"form":        (*builtin.FormData)(nil),
	"queryEscape": builtin.QueryEscape,

	// regexp
	"Regexp": reflect.TypeOf(builtin.Regexp{}),
	"regexp": builtin.RegExp,

	// sort
	"reverse": builtin.Reverse,
	"sort":    builtin.Sort,

	// strconv
	"formatFloat": builtin.FormatFloat,
	"formatInt":   builtin.FormatInt,
	"parseFloat":  builtin.ParseFloat,
	"parseInt":    builtin.ParseInt,

	// strings
	"abbreviate":    builtin.Abbreviate,
	"capitalize":    builtin.Capitalize,
	"capitalizeAll": builtin.CapitalizeAll,
	"hasPrefix":     builtin.HasPrefix,
	"hasSuffix":     builtin.HasSuffix,
	"index":         builtin.Index,
	"indexAny":      builtin.IndexAny,
	"join":          builtin.Join,
	"lastIndex":     builtin.LastIndex,
	"replace":       builtin.Replace,
	"replaceAll":    builtin.ReplaceAll,
	"runeCount":     builtin.RuneCount,
	"split":         builtin.Split,
	"splitAfter":    builtin.SplitAfter,
	"splitAfterN":   builtin.SplitAfterN,
	"splitN":        builtin.SplitN,
	"sprint":        builtin.Sprint,
	"sprintf":       builtin.Sprintf,
	"toKebab":       builtin.ToKebab,
	"toLower":       builtin.ToLower,
	"toUpper":       builtin.ToUpper,
	"trim":          builtin.Trim,
	"trimLeft":      builtin.TrimLeft,
	"trimPrefix":    builtin.TrimPrefix,
	"trimRight":     builtin.TrimRight,
	"trimSuffix":    builtin.TrimSuffix,

	// time
	"Duration":      reflect.TypeOf(builtin.Duration(0)),
	"Hour":          time.Hour,
	"Microsecond":   time.Microsecond,
	"Millisecond":   time.Millisecond,
	"Minute":        time.Minute,
	"Nanosecond":    time.Nanosecond,
	"Second":        time.Second,
	"Time":          reflect.TypeOf(builtin.Time{}),
	"date":          builtin.Date,
	"now":           builtin.Now,
	"parseDuration": builtin.ParseDuration,
	"parseTime":     builtin.ParseTime,
	"unixTime":      builtin.UnixTime,

	// unsafeconv
	"unsafeconv": builtin.Unsafeconv,
}
