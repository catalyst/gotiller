package util

import "fmt"

func ToString(i interface{}) string {
    if i == nil {
        return ""
    }
    return fmt.Sprintf("%v", i)
}
