package util

import (
    "hash/crc32"
    "math/rand"
)

func Sequence(start, length int) []int {
    seq := make([]int, length)
    for i, _ := range seq {
        seq[i] = start + i
    }
    return seq
}

// seed is not seed at all
func TimeOffset(seed string) int {
    if seed == "" {
        return rand.Intn(60)
    }
    return int(crc32.ChecksumIEEE( []byte(seed) ) % 60)
}
