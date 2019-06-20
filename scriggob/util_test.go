// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "testing"

func Test_nextGoVersion(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{
			current: "go1.12",
			want:    "go1.13",
		},
		{
			current: "go1.8",
			want:    "go1.9",
		},
		{
			current: "go1.20.1",
			want:    "go1.21",
		},
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			if got := nextGoVersion(tt.current); got != tt.want {
				t.Errorf("nextGoVersion(%s) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}

func Test_goBaseVersion(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{
			current: "go1.12",
			want:    "go1.12",
		},
		{
			current: "go1.8",
			want:    "go1.8",
		},
		{
			current: "go1.20.1",
			want:    "go1.20",
		},
	}
	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			if got := goBaseVersion(tt.current); got != tt.want {
				t.Errorf("goBaseVersion(%s) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}
