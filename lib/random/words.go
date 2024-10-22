package random

import "github.com/go-loremipsum/loremipsum"

var gen = loremipsum.New()

func Word() string {
	return gen.Word()
}

func Words(r []int) string {
	name := ""
	n := Value(r)
	for x := 0; x < n; x++ {
		if x != 0 {
			name += " "
		}
		name += Word()
	}
	return name
}

func Sentences(r []int) string {
	return gen.Sentences(Value(r))
}
