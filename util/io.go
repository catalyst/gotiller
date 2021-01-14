package util

import (
    "os"
    "time"
    "fmt"
    "io/ioutil"
    "bufio"
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

func SlurpFileAsLines(path string) []string {
    in_f, err := os.Open(path)
    if err != nil {
        panic(err)
    }

    scanner := bufio.NewScanner(in_f)
    var result []string
    for scanner.Scan() {
        result = append(result, scanner.Text())
    }
    return result
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

func Touch(path string) {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        file, err := os.Create(path)
        if err != nil {
            panic(err)
        }
        if err := file.Close(); err != nil {
            panic(err)
        }
    } else {
        now := time.Now().Local()
        if err := os.Chtimes(path, now, now); err != nil {
            panic(err)
        }
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
