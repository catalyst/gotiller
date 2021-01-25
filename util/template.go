package util

import (
    "fmt"
    "hash/crc32"
)

func Sequence(start, length int) []int {
    seq := make([]int, length)
    for i, _ := range seq {
        seq[i] = start + i
    }
    return seq
}

func Hash(in string) string {
    return fmt.Sprintf("%08x", crc32.ChecksumIEEE( []byte(in) ))
}
