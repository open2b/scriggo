// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"fmt"
	"time"

	"github.com/open2b/scriggo/templates"
)

// A Time represents an instant in time.
//
// It is a simplified version of the time.Time type of Go, suitable for use in
// a template.
type Time struct {
	t time.Time
	f TimeFormatter
}

// NewTime returns a Time value at the time of t, formatted using the
// formatter f. It is not intended to be used as a builtin but it can be used
// to implement builtins that return time values.
//
// To format time values as the time package of Go does, use nil as formatter
//
//    NewTime(t, nil)
//
// For example, a builtin that returns the current local time rounded to
// milliseconds and with location "America/New_York" can be implemented as
//
//    func now() builtin.Time {
//        l := time.LoadLocation("America/New_York")
//        t := time.Now().Rounded(time.Millisecond).In(l)
//        return builtin.NewTime(t, nil)
//    }
//
// and once added to the declarations
//
//    templates.Declarations{
//        ...
//        "now" : now,
//        ...
//    }
//
// can be used in a template
//
//    Current time: {{ now() }}
//
func NewTime(t time.Time, f TimeFormatter) Time {
	return Time{t, f}
}

// TimeArguments returns the time.Time and the formatter used as arguments for
// NewTime when t was created. It is not intended to be used as a builtin.
func TimeArguments(t Time) (time.Time, TimeFormatter) {
	return t.t, t.f
}

// TimeFormatter represents time formatters. It is not intended to be used as
// a builtin but for time formatter to pass to the NewTime function.
type TimeFormatter interface {
	// Format returns a textual representation of t formatted according to layout.
	// Format is called by the Time.Format method with a non-empty layout and by
	// the Time.String method with an empty layout.
	Format(t time.Time, layout string) string
}

// AddClock returns the time corresponding to adding the given number of
// hours, minutes and seconds to t. For example, AddClock(1, 30, 15) applied
// to January 1, 2011 14:10:20 returns January 1, 2011 15:40:35.
//
// AddClock normalizes its result in the same way that the Date function does.
func (t Time) AddClock(hours, minutes, seconds int) Time {
	d := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	return Time{t.t.Add(d), t.f}
}

// AddFraction returns the time corresponding to adding the given number of
// milliseconds, microseconds and nanoseconds to t.
//
// AddFraction normalizes its result in the same way that the Date function
// does.
func (t Time) AddFraction(msec, usec, nsec int) Time {
	d := time.Duration(msec)*time.Millisecond + time.Duration(usec)*time.Microsecond + time.Duration(nsec)*time.Nanosecond
	return Time{t.t.Add(d), t.f}
}

// AddDate returns the time corresponding to adding the given number of years,
// months and days to t. For example, AddDate(-1, 2, 3) applied to
// January 1, 2011 returns March 4, 2010.
//
// AddDate normalizes its result in the same way that the Date function does.
func (t Time) AddDate(years, months, days int) Time {
	return Time{t.t.AddDate(years, months, days), t.f}
}

// After reports whether the time instant t is after u.
func (t Time) After(u Time) bool {
	return t.t.After(u.t)
}

// Before reports whether the time instant t is before u.
func (t Time) Before(u Time) bool {
	return t.t.Before(u.t)
}

// Clock returns the hour, minute and second within the day specified by t.
// hour is in the range [0, 23], minute and second are in the range [0, 59].
func (t Time) Clock() (hour, minute, second int) {
	return t.t.Clock()
}

// Date returns the year, month and day in which t occurs. month is in the
// range [1, 12] and day is in the range [1, 31].
func (t Time) Date() (year, month, day int) {
	y, m, d := t.t.Date()
	return y, int(m), d
}

// Day returns the day of the month specified by t, in the range [1, 31]
func (t Time) Day() int {
	return t.t.Day()
}

// Equal reports whether t and u represent the same time instant.
func (t Time) Equal(u Time) bool {
	return t.t.Equal(u.t)
}

// Format formats the time value formatted according to layout using the t's
// formatter.
//
// As a special case, if layout is an empty string, it formats the time as for
// the time.RFC3339Nano layout of Go.
func (t Time) Format(layout string) string {
	if layout == "" {
		return t.t.Format(time.RFC3339Nano)
	}
	if t.f == nil {
		return t.t.Format(layout)
	}
	return t.f.Format(t.t, layout)
}

// Fraction returns the fractional seconds as milliseconds, microseconds and
// nanoseconds within the day specified by t.
//
// For example if the fractional seconds are 0.918273645, it returns 918,
// 918273 and 918273645.
func (t Time) Fraction() (msec, usec, nsec int) {
	d := time.Duration(t.t.Nanosecond())
	return int(d.Milliseconds()), int(d.Microseconds()), int(d.Nanoseconds())
}

// Hour returns the hour within the day specified by t, in the range [0, 23].
func (t Time) Hour() int {
	return t.t.Hour()
}

// JS returns the time as a JavaScript date. The result is undefined if the
// year of t is not in the range [-999999, 999999].
func (t Time) JS() templates.JS {
	y := t.t.Year()
	ms := int64(t.t.Nanosecond()) / int64(time.Millisecond)
	name, offset := t.t.Zone()
	if name == "UTC" {
		format := `new Date("%0.4d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3dZ")`
		if y < 0 || y > 9999 {
			format = `new Date("%+0.6d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3dZ")`
		}
		return templates.JS(fmt.Sprintf(format, y, t.t.Month(), t.t.Day(), t.t.Hour(), t.t.Minute(), t.t.Second(), ms))
	}
	zone := offset / 60
	h, m := zone/60, zone%60
	if m < 0 {
		m = -m
	}
	format := `new Date("%0.4d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3d%+0.2d:%0.2d")`
	if y < 0 || y > 9999 {
		format = `new Date("%+0.6d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3d%+0.2d:%0.2d")`
	}
	return templates.JS(fmt.Sprintf(format, y, t.t.Month(), t.t.Day(), t.t.Hour(), t.t.Minute(), t.t.Second(), ms, h, m))
}

// JSON returns the time as a JSON date.
func (t Time) JSON() templates.JSON {
	return templates.JSON(`"` + t.t.Format(time.RFC3339) + `"`)
}

// Month returns the month of the year specified by t, in the range [1, 12]
// where 1 is January and 12 is December.
func (t Time) Month() int {
	return int(t.t.Month())
}

// Minute returns the minute offset within the hour specified by t, in the range [0, 59].
func (t Time) Minute() int {
	return t.t.Minute()
}

// Second returns the second offset within the minute specified by t, in the range [0, 59].
func (t Time) Second() int {
	return t.t.Second()
}

// String returns the time formatted.
func (t Time) String() string {
	if t.f == nil {
		return t.t.Format(time.RFC3339Nano)
	}
	return t.f.Format(t.t, "")
}

// Sub returns t-u as the hours, minutes, seconds and nanoseconds. minutes and
// seconds are in the range [-59,59] and ns is in the range [0, 999999999].
//
// For example, Sub applied to March 27, 2021 11:21:14.964553 and March 27,
// 2021 10:16:29.370149 returns 1033h 4m 45s 594404000ns.
//
// t-u truncated, in seconds is
//    hours * 3600 + minutes * 60 + seconds
// t-u truncated, in minutes is
//    hours * 60 + minutes
// t-u truncated, in hours is
//    hours
//
func (t Time) Sub(u Time) (hours, minutes, seconds, nsec int) {
	d := t.t.Sub(u.t)
	h, d := d/time.Hour, d%time.Hour
	m, d := d/time.Minute, d%time.Minute
	s, d := d/time.Second, d%time.Second
	return int(h), int(m), int(s), int(d)
}

// UTC returns t with the location set to UTC.
func (t Time) UTC() Time {
	return Time{t.t.UTC(), t.f}
}

// Unix returns t as a Unix time, the number of seconds elapsed since
// January 1, 1970 UTC.
func (t Time) Unix() int64 {
	return t.t.Unix()
}

// Weekday returns the day of the week specified by t, in the range [0, 6]
// where 0 is Sunday and 6 is Saturday.
func (t Time) Weekday() int {
	return int(t.t.Weekday())
}

// Year returns the year in which t occurs.
func (t Time) Year() int {
	return t.t.Year()
}

// timeLayouts contains predefined layouts used by the ParseTime function.
//
// This list of layouts is the same used by Hugo's time function. It is
// copyright 2017 The Hugo Authors and licensed under the Apache license.
// See https://github.com/gohugoio/hugo/blob/master/tpl/time/time.go.
var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05", // iso8601 without timezone
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC850,
	time.ANSIC,
	time.UnixDate,
	time.RubyDate,
	"2006-01-02 15:04:05.999999999 -0700 MST", // Time.String()
	"2006-01-02",
	"02 Jan 2006",
	"2006-01-02T15:04:05-0700", // RFC3339 without timezone hh:mm colon
	"2006-01-02 15:04:05 -07:00",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05Z07:00", // RFC3339 without T
	"2006-01-02 15:04:05Z0700",  // RFC3339 without T or timezone hh:mm colon
	"2006-01-02 15:04:05",
	time.Kitchen,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,
}
