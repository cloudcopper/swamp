package random

import (
	"time"

	"golang.org/x/exp/rand"
)

func init() {
	rand.Seed(uint64(time.Now().UnixNano()))
}

// Value returns random value in range of [a[0],a[1]]
func Value(a []int) int {
	m, n := a[0], a[1]
	return rand.Intn(n-m+1) + m
}

// Element returns random element of a
func Element[T any](a []T) T {
	return a[Value([]int{0, len(a) - 1})]
}

// ByteSlice returns slice of random bytes
func ByteSlice(n int) []byte {
	data := []byte{}
	n = Value([]int{0, n})
	for x := 0; x < n; x++ {
		data = append(data, byte(Value([]int{0, 255})))
	}
	return data
}
