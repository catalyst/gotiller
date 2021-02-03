// Utility functions. To be available in templates.

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

func SafeMatch (s interface{}, re string) bool {
    if s == nil {
        return false
    }

    re_c := regexp.MustCompile(re)
    return re_c.MatchString(s.(string))
}

func SafeReplaceAllString (s interface{}, re string, replacement string) string {
    if s == nil {
        return ""
    }

    re_c := regexp.MustCompile(re)
    return re_c.ReplaceAllString(s.(string), replacement)
}

// For the list of arguments, return first that is not undefined (nil)
func Coalesce(v ...interface{}) interface{} {
    for _, v1 := range v {
        if v1 != nil {
            return v1
        }
    }
    return nil
}

// Splits given string on separator, and joins the bits quoted with "".
// A half-hearted attempt, no escaping.
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

// Give a number between 0 and 60 that somehow represents the given string.
// seed is not seed at all.
func TimeOffset(seed string) int {
    if seed == "" {
        return rand.Intn(60)
    }
    return int(crc32.ChecksumIEEE( []byte(seed) ) % 60)
}
