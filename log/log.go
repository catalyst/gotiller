package log

import std_log "log"

const Ldebug = 1 << 15

type Logger struct {
    *std_log.Logger
}

func New() *Logger {
    return &Logger{ std_log.New(std_log.Writer(), "", 0) }
}

func (l *Logger) SetDebug(dbg bool) bool {
    flags := l.Flags()
    if dbg {
        l.SetFlags(flags | Ldebug)
    } else {
        l.SetFlags(flags &^ Ldebug)
    }
    return (flags & Ldebug) != 0  // old value
}

func (l *Logger) shouldDebug() bool { return (l.Flags() & Ldebug) != 0 }

func (l *Logger) Debug(v ...interface{}) {
    if l.shouldDebug() {
        l.Print(v...)
    }
}

func (l *Logger) Debugln(v ...interface{}) {
    if l.shouldDebug() {
        l.Println(v...)
    }
}

func (l *Logger) Debugf(format string, v ...interface{}) {
    if l.shouldDebug() {
        l.Printf(format, v...)
    }
}

func (l *Logger) WithDebug(f func ()) {
    dbg := l.SetDebug(true)
    defer func () { l.SetDebug(dbg) }()

    f()
}
