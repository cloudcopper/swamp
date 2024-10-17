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
	min, max := a[0], a[1]
	return rand.Intn(max-min+1) + min
}

// Element returns random element of a
func Element[T any](a []T) T {
	return a[Value([]int{0, len(a) - 1})]
}

// ByteSlice returns slice of random bytes
func ByteSlice(max int) []byte {
	data := []byte{}
	max = Value([]int{0, max})
	for x := 0; x < max; x++ {
		data = append(data, byte(Value([]int{0, 255})))
	}
	return data
}
