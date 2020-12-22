// +build go1.16

// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs

import "io/fs"

type (
	FS         = fs.FS
	File       = fs.File
	ReadFileFS = fs.ReadFileFS
)
