// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"testing"
	"time"
)

var parseTimeTests = []struct {
	value  string
	layout string
}{
	{"2021-03-27T11:21:14.964553+01:00", time.RFC3339},
	{"2021-03-27T11:21:14+01:00", "2006-01-02T15:04:05Z07:00"},
	{"2021-03-27T11:21:14", "2006-01-02T15:04:05"},
	{"Sat, 27 Mar 2021 11:21:14 +0100", "Mon, 02 Jan 2006 15:04:05 -0700"},
	{"Sat, 27 Mar 2021 11:21:14 CET", "Mon, 02 Jan 2006 15:04:05 MST"},
	{"27 Mar 21 11:21 +0100", "02 Jan 06 15:04 -0700"},
	{"27 Mar 21 11:21 CET", "02 Jan 06 15:04 MST"},
	{"Saturday, 27-Mar-21 11:21:14 CET", "Monday, 02-Jan-06 15:04:05 MST"},
	{"Sat Mar 27 11:21:14 2021", "Mon Jan _2 15:04:05 2006"},
	{"Sat Mar 27 11:21:14 CET 2021", "Mon Jan _2 15:04:05 MST 2006"},
	{"Sat Mar 27 11:21:14 +0100 2021", "Mon Jan 02 15:04:05 -0700 2006"},
	{"2021-03-27 11:21:14.964553 +0100 CET", "2006-01-02 15:04:05.999999999 -0700 MST"},
	{"2021-03-27", "2006-01-02"},
	{"27 Mar 2021", "02 Jan 2006"},
	{"2021-03-27T11:21:14+0100", "2006-01-02T15:04:05-0700"},
	{"2021-03-27 11:21:14 +01:00", "2006-01-02 15:04:05 -07:00"},
	{"2021-03-27 11:21:14 +0100", "2006-01-02 15:04:05 -0700"},
	{"2021-03-27 11:21:14+01:00", "2006-01-02 15:04:05Z07:00"},
	{"2021-03-27 11:21:14+0100", "2006-01-02 15:04:05Z0700"},
	{"2021-03-27 11:21:14", "2006-01-02 15:04:05"},
	{"11:21AM", "3:04PM"},
	{"Mar 27 11:21:14", "Jan _2 15:04:05"},
	{"Mar 27 11:21:14.964", "Jan _2 15:04:05"},
	{"Mar 27 11:21:14.964553", "Jan _2 15:04:05"},
	{"Mar 27 11:21:14.964553000", "Jan _2 15:04:05"},
}

func TestParseTime(t *testing.T) {
	for _, cas := range parseTimeTests {
		got, err := ParseTime("", cas.value)
		if err != nil {
			t.Fatalf("source: %q, got error %q\n", cas.value, err)
		}
		expected, err := time.Parse(cas.layout, cas.value)
		if err != nil {
			t.Fatal(err)
		}
		if !expected.Equal(got.t) {
			t.Errorf("source: %q, got %q, expecting %q\n", cas.value, got, expected)
		}
	}
}

var sptime = func(t Time) string { return t.t.Format(time.RFC3339Nano) }

func TestTime(t *testing.T) {
	cet, err := time.LoadLocation("CET")
	if err != nil {
		t.Fatal(err)
	}
	t1 := NewTime(time.Date(2021, 3, 27, 11, 21, 14, 964553000, cet))
	t2 := NewTime(time.Date(2021, 2, 12, 10, 16, 29, 370149000, cet))
	for _, expr := range []struct {
		got      string
		expected string
	}{
		{sptime(t1.Add(2 * time.Hour)), "2021-03-27T13:21:14.964553+01:00"},
		{sptime(t1.AddDate(2, 3, 7)), "2023-07-04T11:21:14.964553+02:00"},
		{spf("%t", t1.After(t2)), "true"},
		{spf("%t", t1.Before(t2)), "false"},
		{spf("%v", func() []int { h, m, s := t1.Clock(); return []int{h, m, s} }()), "[11 21 14]"},
		{spf("%v", func() []int { y, m, d := t1.Date(); return []int{y, int(m), d} }()), "[2021 3 27]"},
		{spf("%d", t1.Day()), "27"},
		{spf("%t", t1.Equal(t1)), "true"},
		{spf("%t", t1.Equal(t2)), "false"},
		{spf("%s", t1.Format("Monday, 02-Jan-06 15:04:05 MST")), "Saturday, 27-Mar-21 11:21:14 CET"},
		{spf("%d", t1.Hour()), "11"},
		{spf("%t", t1.IsZero()), "false"},
		{spf("%t", time.Time{}.IsZero()), "true"},
		{spf("%s", t1.JS()), `new Date("2021-03-27T11:21:14.964+01:00")`},
		{spf("%s", t1.JSON()), `"2021-03-27T11:21:14+01:00"`},
		{spf("%d", t1.Minute()), "21"},
		{spf("%d", t1.Month()), "3"},
		{spf("%d", t1.Nanosecond()), "964553000"},
		{spf("%s", t1.Round(3*time.Hour+25*time.Second)), "2021-03-27 12:24:35 +0100 CET"},
		{spf("%d", t1.Second()), "14"},
		{spf("%s", t1.String()), "2021-03-27 11:21:14.964553 +0100 CET"},
		{spf("%s", t1.Sub(t2)), "1033h4m45.594404s"},
		{spf("%s", t1.Truncate(3*time.Hour+25*time.Second)), "2021-03-27 09:24:10 +0100 CET"},
		{spf("%s", t1.UTC()), "2021-03-27 10:21:14.964553 +0000 UTC"},
		{spf("%d", t1.Unix()), "1616840474"},
		{spf("%d", t1.UnixNano()), "1616840474964553000"},
		{spf("%d", t1.Weekday()), "6"},
		{spf("%d", t1.Year()), "2021"},
		{spf("%d", t1.YearDay()), "86"},
	} {
		if expr.got != expr.expected {
			t.Errorf("got %q, expecting %q\n", expr.got, expr.expected)
		}
	}
}
