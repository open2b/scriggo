// Go version: go1.11.5

package syslog

import original "log/syslog"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Dial": original.Dial,
	"LOG_ALERT": scrigo.Constant(original.LOG_ALERT, nil),
	"LOG_AUTH": scrigo.Constant(original.LOG_AUTH, nil),
	"LOG_AUTHPRIV": scrigo.Constant(original.LOG_AUTHPRIV, nil),
	"LOG_CRIT": scrigo.Constant(original.LOG_CRIT, nil),
	"LOG_CRON": scrigo.Constant(original.LOG_CRON, nil),
	"LOG_DAEMON": scrigo.Constant(original.LOG_DAEMON, nil),
	"LOG_DEBUG": scrigo.Constant(original.LOG_DEBUG, nil),
	"LOG_EMERG": scrigo.Constant(original.LOG_EMERG, nil),
	"LOG_ERR": scrigo.Constant(original.LOG_ERR, nil),
	"LOG_FTP": scrigo.Constant(original.LOG_FTP, nil),
	"LOG_INFO": scrigo.Constant(original.LOG_INFO, nil),
	"LOG_KERN": scrigo.Constant(original.LOG_KERN, nil),
	"LOG_LOCAL0": scrigo.Constant(original.LOG_LOCAL0, nil),
	"LOG_LOCAL1": scrigo.Constant(original.LOG_LOCAL1, nil),
	"LOG_LOCAL2": scrigo.Constant(original.LOG_LOCAL2, nil),
	"LOG_LOCAL3": scrigo.Constant(original.LOG_LOCAL3, nil),
	"LOG_LOCAL4": scrigo.Constant(original.LOG_LOCAL4, nil),
	"LOG_LOCAL5": scrigo.Constant(original.LOG_LOCAL5, nil),
	"LOG_LOCAL6": scrigo.Constant(original.LOG_LOCAL6, nil),
	"LOG_LOCAL7": scrigo.Constant(original.LOG_LOCAL7, nil),
	"LOG_LPR": scrigo.Constant(original.LOG_LPR, nil),
	"LOG_MAIL": scrigo.Constant(original.LOG_MAIL, nil),
	"LOG_NEWS": scrigo.Constant(original.LOG_NEWS, nil),
	"LOG_NOTICE": scrigo.Constant(original.LOG_NOTICE, nil),
	"LOG_SYSLOG": scrigo.Constant(original.LOG_SYSLOG, nil),
	"LOG_USER": scrigo.Constant(original.LOG_USER, nil),
	"LOG_UUCP": scrigo.Constant(original.LOG_UUCP, nil),
	"LOG_WARNING": scrigo.Constant(original.LOG_WARNING, nil),
	"New": original.New,
	"NewLogger": original.NewLogger,
	"Priority": reflect.TypeOf(original.Priority(int(0))),
	"Writer": reflect.TypeOf(original.Writer{}),
}
