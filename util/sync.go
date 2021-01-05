package util

import (
    "sync"
)

type WaitGroup struct {
    sync.WaitGroup
    error_ch chan interface{}
}
func (wg *WaitGroup) init() {
    wg.error_ch = make(chan interface{})

    go wg.collect_errors()
}
func (wg *WaitGroup) collect_errors() {
    var errs []interface{}
    for err:= range wg.error_ch {
        errs = append(errs, err)
    }
    if errs != nil {
        panic(errs)
    }
}

func NewWaitGroup() *WaitGroup {
    wg := new(WaitGroup)
    wg.init()
    return wg
}

func (wg *WaitGroup) Run(f func()) {
    wg.Add(1)
    go func() {
        defer func() {
            wg.Done()

            if r := recover(); r != nil {
                wg.error_ch <- r
            }
        }()

        f()
    }()
}

func (wg *WaitGroup) Wait() {
    wg.WaitGroup.Wait()
    close(wg.error_ch)
}
