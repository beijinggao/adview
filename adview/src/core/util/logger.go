package util

//
// 日志模块
// 封装了glog实现Debug,Info,Warn,Error,Fatal,Flush函数
//

import (
	"github.com/golang/glog"
)

func Debug(format string, args ...interface{}) {
	glog.V(3).Infof(format, args...)
}

func Info(format string, args ...interface{}) {
	glog.V(0).Infof(format, args...)
}

func Warn(format string, args ...interface{}) {
	glog.Warningf(format, args...)
}

func Error(format string, args ...interface{}) {
	glog.Errorf(format, args...)
}

func Fatal(format string, args ...interface{}) {
	glog.Fatalf(format, args...)
}

func Flush() {
	glog.Flush()
}
