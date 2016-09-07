package taskpool

import (
	"github.com/antlinker/alog"
)

const (
	logTag = "TASKPOOL"
)

// Tlog 日志模块
var Tlog = alog.NewALog()

func init() {
	Tlog.SetLogTag(logTag)
	Tlog.SetEnabled(false)
}

// TlogInit 日志初始化
func TlogInit(configs string) {
	Tlog.ReloadConfig(configs)

	Tlog.SetLogTag(logTag)
}
