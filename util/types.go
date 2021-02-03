// Utility functions. Basic conversions.

package util

import (
    "fmt"
    "strconv"
)

func ToString(i interface{}) string {
    if i == nil {
        return ""
    }
    return fmt.Sprintf("%v", i)
}

func AtoI(s string) int {
    i, err := strconv.Atoi(s)
    if err != nil {
        panic(err)
    }
    return i
}
