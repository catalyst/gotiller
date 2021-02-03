// Utility functions. io and filesystem related

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

func ReadDir(path string) []os.FileInfo {
    dir_entries, err := ioutil.ReadDir(path);
    if err != nil {
        panic(err)
    }
    return dir_entries
}

func IsFile(path string) bool {
    stat, err := os.Stat(path)

    if err != nil {
        if os.IsNotExist(err) {
            return false
        }

        panic(err)
    }

    if stat.IsDir() {
        panic(path + " is a directory")
    }

    return true
}

func Touch(path string) {
    if IsFile(path) {
        now := time.Now().Local()
        if err := os.Chtimes(path, now, now); err != nil {
            panic(err)
        }
    } else {
        file, err := os.Create(path)
        if err != nil {
            panic(err)
        }
        if err := file.Close(); err != nil {
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

func ResolveLink(path string) string {
    stat, err := os.Lstat(path)
    if err != nil {
        panic(err)
    }
    if stat.Mode() & os.ModeSymlink != 0 {
        if l_path, err := os.Readlink(path); err != nil {
            if filepath.IsAbs(l_path) {
                return l_path
            }
            return filepath.Join(filepath.Dir(path), l_path)
        }
        panic(err)
    }
    return path
}
