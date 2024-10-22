package random

import (
	"fmt"
	"strings"
)

// Declare return fake equivalent of shell
// export output with 'declare -x'
func Declare(n int) string {
	str := ""
	n = Value([]int{1, n})

	for x := 0; x < n; x++ {
		name := strings.ReplaceAll(Words([]int{1, 3}), " ", "_")
		value := Words([]int{1, 5})
		str += fmt.Sprintf("declare -x %s=%q\n", name, value)
	}

	return str
}
