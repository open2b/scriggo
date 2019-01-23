// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"reflect"
	"time"
)

var _time = Package{
	"ANSIC":                  time.ANSIC,
	"UnixDate":               time.UnixDate,
	"RubyDate":               time.RubyDate,
	"RFC822":                 time.RFC822,
	"RFC822Z":                time.RFC822Z,
	"RFC850":                 time.RFC850,
	"RFC1123":                time.RFC1123,
	"RFC1123Z":               time.RFC1123Z,
	"RFC3339":                time.RFC3339,
	"RFC3339Nano":            time.RFC3339Nano,
	"Kitchen":                time.Kitchen,
	"Stamp":                  time.Stamp,
	"StampMilli":             time.StampMilli,
	"StampMicro":             time.StampMicro,
	"StampNano":              time.StampNano,
	"Nanosecond":             time.Nanosecond,
	"Microsecond":            time.Microsecond,
	"Millisecond":            time.Millisecond,
	"Second":                 time.Second,
	"Minute":                 time.Minute,
	"Hour":                   time.Hour,
	"January":                time.January,
	"February":               time.February,
	"March":                  time.March,
	"April":                  time.April,
	"May":                    time.May,
	"June":                   time.June,
	"July":                   time.July,
	"August":                 time.August,
	"September":              time.September,
	"October":                time.October,
	"November":               time.November,
	"December":               time.December,
	"Sunday":                 time.Sunday,
	"Monday":                 time.Monday,
	"Tuesday":                time.Tuesday,
	"Wednesday":              time.Wednesday,
	"Thursday":               time.Thursday,
	"Friday":                 time.Friday,
	"Saturday":               time.Saturday,
	"After":                  time.After,
	"Sleep":                  time.Sleep,
	"Tick":                   time.Tick,
	"ParseDuration":          time.ParseDuration,
	"Since":                  time.Since,
	"Until":                  time.Until,
	"FixedZone":              time.FixedZone,
	"LoadLocation":           time.LoadLocation,
	"LoadLocationFromTZData": time.LoadLocationFromTZData,
	"NewTicker":              time.NewTicker,
	"Date":                   time.Date,
	"Now":                    time.Now,
	"Parse":                  time.Parse,
	"ParseInLocation":        time.ParseInLocation,
	"Unix":                   time.Unix,
	"AfterFunc":              time.AfterFunc,
	"NewTimer":               time.NewTimer,
	"Duration":               reflect.TypeOf(time.Duration(0)),
	"Location":               reflect.TypeOf(time.Location{}),
	"Month":                  reflect.TypeOf(time.Month(0)),
	"Weekday":                reflect.TypeOf(time.Weekday(0)),
	"Timer":                  reflect.TypeOf(time.Timer{}),
	"Ticker":                 reflect.TypeOf(time.Ticker{}),
	"Time":                   reflect.TypeOf(time.Time{}),
	"ParseError":             reflect.TypeOf(time.ParseError{}),
}
