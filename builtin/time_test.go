// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"testing"
	"time"
)

var parseTimeTests = []struct {
	value    string
	expected string
}{
	{"2021-03-27T11:21:14.964553+01:00", "2021-03-27T11:21:14.964553+01:00"},
	{"2021-03-27T11:21:14+01:00", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27T11:21:14", "2021-03-27T11:21:14Z"},
	{"Sat, 27 Mar 2021 11:21:14 +0100", "2021-03-27T11:21:14+01:00"},
	{"Sat, 27 Mar 2021 11:21:14 CET", "2021-03-27T11:21:14+01:00"},
	{"27 Mar 21 11:21 +0100", "2021-03-27T11:21:00+01:00"},
	{"27 Mar 21 11:21 CET", "2021-03-27T11:21:00+01:00"},
	{"Saturday, 27-Mar-21 11:21:14 CET", "2021-03-27T11:21:14+01:00"},
	{"Sat Mar 27 11:21:14 2021", "2021-03-27T11:21:14Z"},
	{"Sat Mar 27 11:21:14 CET 2021", "2021-03-27T11:21:14+01:00"},
	{"Sat Mar 27 11:21:14 +0100 2021", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14.964553 +0100 CET", "2021-03-27T11:21:14.964553+01:00"},
	{"2021-03-27", "2021-03-27T00:00:00Z"},
	{"27 Mar 2021", "2021-03-27T00:00:00Z"},
	{"2021-03-27T11:21:14+0100", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14 +01:00", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14 +0100", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14+01:00", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14+0100", "2021-03-27T11:21:14+01:00"},
	{"2021-03-27 11:21:14", "2021-03-27T11:21:14Z"},
	{"11:21AM", "0000-01-01T11:21:00Z"},
	{"Mar 27 11:21:14", "0000-03-27T11:21:14Z"},
	{"Mar 27 11:21:14.964", "0000-03-27T11:21:14.964Z"},
	{"Mar 27 11:21:14.964553", "0000-03-27T11:21:14.964553Z"},
	{"Mar 27 11:21:14.964553000", "0000-03-27T11:21:14.964553Z"},
}

func TestParseTime(t *testing.T) {
	for _, cas := range parseTimeTests {
		tm, err := ParseTime("", cas.value)
		if err != nil {
			t.Fatalf("source: %q, got error %q, expecting %q\n", cas.value, err, cas.expected)
		}
		if got := tm.t.Format(time.RFC3339Nano); got != cas.expected {
			t.Errorf("source: %q, got %q, expecting %q\n", cas.value, got, cas.expected)
		}
	}
}

var t1, _ = ParseTime(time.RFC3339Nano, "2021-03-27T11:21:14.964553+01:00")
var t2, _ = ParseTime(time.RFC3339Nano, "2021-02-12T10:16:29.370149+01:00")

var sptime = func(t Time) string { return t.t.Format(time.RFC3339Nano) }

var timeTests = []struct {
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
}

func TestTime(t *testing.T) {
	for _, expr := range timeTests {
		if expr.got != expr.expected {
			t.Errorf("got %q, expecting %q\n", expr.got, expr.expected)
		}
	}
}
