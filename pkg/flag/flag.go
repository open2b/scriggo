// Go version: go1.11.5

package flag

import original "flag"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Arg": original.Arg,
	"Args": original.Args,
	"Bool": original.Bool,
	"BoolVar": original.BoolVar,
	"CommandLine": &original.CommandLine,
	"ContinueOnError": scrigo.Constant(original.ContinueOnError, nil),
	"Duration": original.Duration,
	"DurationVar": original.DurationVar,
	"ErrHelp": &original.ErrHelp,
	"ErrorHandling": reflect.TypeOf(original.ErrorHandling(int(0))),
	"ExitOnError": scrigo.Constant(original.ExitOnError, nil),
	"Flag": reflect.TypeOf(original.Flag{}),
	"FlagSet": reflect.TypeOf(original.FlagSet{}),
	"Float64": original.Float64,
	"Float64Var": original.Float64Var,
	"Getter": reflect.TypeOf((*original.Getter)(nil)).Elem(),
	"Int": original.Int,
	"Int64": original.Int64,
	"Int64Var": original.Int64Var,
	"IntVar": original.IntVar,
	"Lookup": original.Lookup,
	"NArg": original.NArg,
	"NFlag": original.NFlag,
	"NewFlagSet": original.NewFlagSet,
	"PanicOnError": scrigo.Constant(original.PanicOnError, nil),
	"Parse": original.Parse,
	"Parsed": original.Parsed,
	"PrintDefaults": original.PrintDefaults,
	"Set": original.Set,
	"String": original.String,
	"StringVar": original.StringVar,
	"Uint": original.Uint,
	"Uint64": original.Uint64,
	"Uint64Var": original.Uint64Var,
	"UintVar": original.UintVar,
	"UnquoteUsage": original.UnquoteUsage,
	"Usage": &original.Usage,
	"Value": reflect.TypeOf((*original.Value)(nil)).Elem(),
	"Var": original.Var,
	"Visit": original.Visit,
	"VisitAll": original.VisitAll,
}
