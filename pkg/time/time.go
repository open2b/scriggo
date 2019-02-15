// Go version: go1.11.5

package time

import original "time"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"After": original.After,
	"AfterFunc": original.AfterFunc,
	"Date": original.Date,
	"Duration": reflect.TypeOf(original.Duration(int64(0))),
	"FixedZone": original.FixedZone,
	"LoadLocation": original.LoadLocation,
	"LoadLocationFromTZData": original.LoadLocationFromTZData,
	"Local": &original.Local,
	"Location": reflect.TypeOf(original.Location{}),
	"Month": reflect.TypeOf(original.Month(int(0))),
	"NewTicker": original.NewTicker,
	"NewTimer": original.NewTimer,
	"Now": original.Now,
	"Parse": original.Parse,
	"ParseDuration": original.ParseDuration,
	"ParseError": reflect.TypeOf(original.ParseError{}),
	"ParseInLocation": original.ParseInLocation,
	"Since": original.Since,
	"Sleep": original.Sleep,
	"Tick": original.Tick,
	"Ticker": reflect.TypeOf(original.Ticker{}),
	"Time": reflect.TypeOf(original.Time{}),
	"Timer": reflect.TypeOf(original.Timer{}),
	"UTC": &original.UTC,
	"Unix": original.Unix,
	"Until": original.Until,
	"Weekday": reflect.TypeOf(original.Weekday(int(0))),
}
