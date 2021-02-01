package util

import (
    "strings"
    "regexp"
    "hash/crc32"
    "math/rand"
)

func SafeValue(s interface{}) string {
    if s == nil {
        return ""
    }
    return s.(string)
}

func SafeToLower(s interface{}) string {
    return strings.ToLower( SafeValue(s) )
}

func SafeReplaceAllString (s interface{}, re string, replacement string) string {
    if s == nil {
        return ""
    }

    re_c := regexp.MustCompile(re)
    return re_c.ReplaceAllString(s.(string), replacement)
}

func Coalesce(v ...interface{}) interface{} {
    for _, v1 := range v {
        if v1 != nil {
            return v1
        }
    }
    return nil
}

func QuotedList(l interface{}, separator string) string {
    if l == nil {
        return ""
    }

    return `"` + strings.Join( strings.Split(l.(string), separator), `", "` ) + `"`
}

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
