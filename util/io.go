package util

import (
    "os"
    "fmt"
    "io/ioutil"
    "path/filepath"
)

func SlurpFile(path string) []byte {
    in_f, err := os.Open(path)
    if err != nil {
        panic(err)
    }

    bytes, err := ioutil.ReadAll(in_f)
    if err != nil {
        panic(err)
    }

    return bytes
}

func WriteFile(path string, content []byte) {
    if err := ioutil.WriteFile(path, content, os.FileMode(0644)); err != nil {
        panic(err)
    }
}

func Mkdir(path string) {
    if info, err := os.Stat(path); err == nil {
        if info.IsDir() {
            return
        }
        panic(path + " is not a dir")
    }

    if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
        panic(err)
    }
}

func PrintDirTree(root string) error {
    return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        fmt.Println(path, info.Size())
        return nil
    })
}
