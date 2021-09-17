// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"fmt"
	"time"

	"github.com/open2b/scriggo/native"
)

// A Duration represents the elapsed time between two instants
// as an int64 nanosecond count. The representation limits the
// largest representable duration to approximately 290 years.
type Duration = time.Duration

// A Time represents an instant in time.
//
// It is a stripped down version of the Go time.Time type, with additional
// methods to show time values in JavaScript and JSON contexts.
type Time struct {
	t time.Time
}

// NewTime returns a Time value at the time of t. It is not intended to be
// used as a builtin but it can be used to implement builtins that return time
// values.
//
// For example, a builtin that returns the current local time rounded to
// milliseconds and with location "America/New_York" can be implemented as
//
//    func now() builtin.Time {
//        l := time.LoadLocation("America/New_York")
//        t := time.Now().Rounded(time.Millisecond).In(l)
//        return builtin.NewTime(t)
//    }
//
// and once added to the declarations
//
//    native.Declarations{
//        ...
//        "now" : now,
//        ...
//    }
//
// it can be used in a template
//
//    Current time: {{ now() }}
//
func NewTime(t time.Time) Time {
	return Time{t}
}

// Add returns the time t+d.
func (t Time) Add(d Duration) Time {
	return Time{t.t.Add(d)}
}

// AddDate returns the time corresponding to adding the given number of years,
// months and days to t. For example, AddDate(-1, 2, 3) applied to January 1,
// 2011 returns March 4, 2010.
//
// AddDate normalizes its result in the same way that the Date function does.
func (t Time) AddDate(years, months, days int) Time {
	return Time{t.t.AddDate(years, months, days)}
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

// Day returns the day of the month specified by t, in the range [1, 31].
func (t Time) Day() int {
	return t.t.Day()
}

// Equal reports whether t and u represent the same time instant.
func (t Time) Equal(u Time) bool {
	return t.t.Equal(u.t)
}

// Format returns a textual representation of the time value formatted
// according to layout, which defines the format by showing how the reference
// time, defined to be
//	Mon Jan 2 15:04:05 -0700 MST 2006
// would be displayed if it were the value; it serves as an example of the
// desired output. The same display rules will then be applied to the time
// value.
//
// A fractional second is represented by adding a period and zeros
// to the end of the seconds section of layout string, as in "15:04:05.000"
// to format a time stamp with millisecond precision.
func (t Time) Format(layout string) string {
	return t.t.Format(layout)
}

// Hour returns the hour within the day specified by t, in the range [0, 23].
func (t Time) Hour() int {
	return t.t.Hour()
}

// IsZero reports whether t represents the zero time instant,
// January 1, year 1, 00:00:00 UTC.
func (t Time) IsZero() bool {
	return t.t.IsZero()
}

// JS returns the time as a JavaScript date. The result is undefined if the
// year of t is not in the range [-999999, 999999].
func (t Time) JS() native.JS {
	y := t.t.Year()
	ms := int64(t.t.Nanosecond()) / int64(time.Millisecond)
	name, offset := t.t.Zone()
	if name == "UTC" {
		format := `new Date("%0.4d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3dZ")`
		if y < 0 || y > 9999 {
			format = `new Date("%+0.6d-%0.2d-%0.2dT%0.2d:%0.2d:%0.2d.%0.3dZ")`
		}
		return native.JS(fmt.Sprintf(format, y, t.t.Month(), t.t.Day(), t.t.Hour(), t.t.Minute(), t.t.Second(), ms))
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
	return native.JS(fmt.Sprintf(format, y, t.t.Month(), t.t.Day(), t.t.Hour(), t.t.Minute(), t.t.Second(), ms, h, m))
}

// JSON returns a time in a format suitable for use in JSON.
func (t Time) JSON() native.JSON {
	return native.JSON(`"` + t.t.Format(time.RFC3339) + `"`)
}

// Month returns the month of the year specified by t, in the range [1, 12]
// where 1 is January and 12 is December.
func (t Time) Month() int {
	return int(t.t.Month())
}

// Minute returns the minute offset within the hour specified by t, in the
// range [0, 59].
func (t Time) Minute() int {
	return t.t.Minute()
}

// Nanosecond returns the nanosecond offset within the second specified by t,
// in the range [0, 999999999].
func (t Time) Nanosecond() int {
	return t.t.Nanosecond()
}

// Round returns the result of rounding t to the nearest multiple of d (since
// the zero time). The rounding behavior for halfway values is to round up.
// If d <= 0, Round returns t stripped of any monotonic clock reading but
// otherwise unchanged.
//
// Round operates on the time as an absolute duration since the zero time; it
// does not operate on the presentation form of the time. Thus, Round(Hour)
// may return a time with a non-zero minute, depending on the time's Location.
func (t Time) Round(d Duration) Time {
	return Time{t.t.Round(d)}
}

// Second returns the second offset within the minute specified by t, in the
// range [0, 59].
func (t Time) Second() int {
	return t.t.Second()
}

// String returns the time formatted.
func (t Time) String() string {
	return t.t.String()
}

// Sub returns the duration t-u. If the result exceeds the maximum (or minimum)
// value that can be stored in a Duration, the maximum (or minimum) duration
// will be returned.
// To compute t-d for a duration d, use t.Add(-d).
func (t Time) Sub(u Time) Duration {
	return t.t.Sub(u.t)
}

// Truncate returns the result of rounding t down to a multiple of d (since
// the zero time). If d <= 0, Truncate returns t stripped of any monotonic
// clock reading but otherwise unchanged.
//
// Truncate operates on the time as an absolute duration since the zero time;
// it does not operate on the presentation form of the time. Thus,
// Truncate(Hour) may return a time with a non-zero minute, depending on the
// time's Location.
func (t Time) Truncate(d Duration) Time {
	return Time{t.t.Truncate(d)}
}

// UTC returns t with the location set to UTC.
func (t Time) UTC() Time {
	return Time{t.t.UTC()}
}

// Unix returns t as a Unix time, the number of seconds elapsed since January
// 1, 1970 UTC. The result does not depend on the location associated with t.
// Unix-like operating systems often record time as a 32-bit count of seconds,
// but since the method here returns a 64-bit value it is valid for billions
// of years into the past or future.
func (t Time) Unix() int64 {
	return t.t.Unix()
}

// UnixNano returns t as a Unix time, the number of nanoseconds elapsed since
// January 1, 1970 UTC. The result is undefined if the Unix time in
// nanoseconds cannot be represented by an int64 (a date before the year 1678
// or after 2262). Note that this means the result of calling UnixNano on the
// zero Time is undefined. The result does not depend on the location
// associated with t.
func (t Time) UnixNano() int64 {
	return t.t.UnixNano()
}

// Weekday returns the day of the week specified by t.
func (t Time) Weekday() int {
	return int(t.t.Weekday())
}

// Year returns the year in which t occurs.
func (t Time) Year() int {
	return t.t.Year()
}

// YearDay returns the day of the year specified by t, in the range [1, 365]
// for non-leap years, and [1, 366] in leap years.
func (t Time) YearDay() int {
	return t.t.YearDay()
}
