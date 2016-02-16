package taskpool

import (
	"github.com/antlinker/alog"
)

const (
	LOG_TAG = "TASKPOOL"
)

var Tlog *alog.ALog = alog.NewALog()

func init() {
	Tlog.SetLogTag(LOG_TAG)
	Tlog.SetEnabled(false)
}

func TlogInit(configs ...string) {
	Tlog = alog.NewALog(configs...)
	Tlog.SetLogTag(LOG_TAG)
}
