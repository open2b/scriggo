// Copyright 2025 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "testing"

func TestParseAddr(t *testing.T) {
	const (
		defaultHost = "localhost"
		defaultPort = "8080"
	)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "hostAndPort",
			input: "example.com:80",
			want:  "example.com:80",
		},
		{
			name:  "hostOnly",
			input: "example.com",
			want:  "example.com" + ":" + defaultPort,
		},
		{
			name:  "portOnly",
			input: ":80",
			want:  defaultHost + ":80",
		},
		{name: "empty", input: "", wantErr: true},
		{name: "colonOnly", input: ":", wantErr: true},
		{name: "zeroPort", input: ":0", wantErr: true},
		{name: "largePort", input: ":99999", wantErr: true},
		{name: "invalidPort", input: "bad:port", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := parseAddr(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got no error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error for %q, got %v", tt.input, err)
			}
			if addr != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, addr)
			}
		})
	}
}
