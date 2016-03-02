package log

import (
"io"
"log"
"os"
"runtime/debug"
)

var (
l *dsLogger = nil
)

const (
LevelFatal = iota
LevelPanic = iota
LevelError = iota
LevelInfo  = iota
LevelDebug = iota
)

const (
prefixDebug = "[DEBUG]"
prefixInfo  = "[INFO]"
prefixError = "[ERROR]"
prefixPanic = "[PANIC]"
prefixFatal = "[FATAL]"
)

type dsLogger struct {
level  int
debugL *log.Logger
infoL  *log.Logger
errorL *log.Logger
panicL *log.Logger
fatalL *log.Logger
}

func InitLog(prefix string, logfileName string, level int) {
if l != nil {
return
}
l = new(dsLogger)
l.level = level
mw := io.MultiWriter(os.Stdout)
if logfileName != "" {
f, err := os.OpenFile(logfileName, os.O_WRONLY|os.O_CREATE, 0666)
if err != nil {
log.Fatal(err)
}
mw = io.MultiWriter(os.Stdout, f)
}
if level >= LevelDebug {
l.debugL = log.New(mw, prefix+prefixDebug, log.LstdFlags)
}
if level >= LevelInfo {
l.infoL = log.New(mw, prefix+prefixInfo, log.LstdFlags)
}
if level >= LevelError {
l.errorL = log.New(mw, prefix+prefixError, log.LstdFlags)
}
if level >= LevelPanic {
l.panicL = log.New(mw, prefix+prefixPanic, log.LstdFlags)
}
if level >= LevelFatal {
l.fatalL = log.New(mw, prefix+prefixFatal, log.LstdFlags)
}
}

func GetInfoLogger() *log.Logger {
return l.infoL
}
func Debug(v ...interface{}) {
if l == nil {
log.Println(v...)
return
}
if l.level < LevelDebug {
return
}
l.debugL.Println(v...)
}

func Debugf(format string, v ...interface{}) {
if l == nil {
log.Printf(format, v...)
return
}
if l.level < LevelDebug {
return
}
l.debugL.Printf(format, v...)
}

func Info(v ...interface{}) {
if l == nil {
log.Println(v...)
return
}
if l.level < LevelInfo {
return
}
l.infoL.Println(v...)
}

func Infof(format string, v ...interface{}) {
if l == nil {
log.Printf(format, v...)
return
}
if l.level < LevelInfo {
return
}
l.infoL.Printf(format, v...)

}

func Error(v ...interface{}) {
if l == nil {
log.Println(v...)
return
}
l.errorL.Println(v...)
}

func Errorf(format string, v ...interface{}) {
if l == nil {
log.Printf(format, v...)
return
}
l.errorL.Printf(format, v...)
}

func Fatal(v ...interface{}) {
if l == nil {
log.Println(v...)
} else {
l.fatalL.Println(v...)
}
os.Exit(1)
}

func Panic(v interface{}) {
if l == nil {
log.Println(v)
log.Println(string(debug.Stack()))
} else {
l.panicL.Println(v)
l.panicL.Println(string(debug.Stack()))
}
panic(v)
}

func Fatalf(format string, v ...interface{}) {
if l == nil {
log.Printf(format, v...)
} else {
l.fatalL.Printf(format, v...)
}
os.Exit(1)
}
