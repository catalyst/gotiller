package util

import (
    "testing"
    "bytes"

    "github.com/catalyst/gotiller/log"
)

func SupressLogForTest(t *testing.T, l *log.Logger) func() {
    var buff bytes.Buffer
    log_w := l.Writer()
    l.SetOutput(&buff)

    return func() {
        t.Log(string(buff.Bytes()))
        l.SetOutput(log_w)
    }
}
