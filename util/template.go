package util

func Sequence (start, length int) []int {
    seq := make([]int, length)
    for i, _ := range seq {
        seq[i] = start + i
    }
    return seq
}
