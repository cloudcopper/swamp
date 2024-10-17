package random

import (
	"fmt"
	"strings"
)

// Declare return fake equivalent of shell
// export output with 'declare -x'
func Declare(max int) string {
	str := ""
	max = Value([]int{1, max})

	for x := 0; x < max; x++ {
		name := strings.ReplaceAll(Words([]int{1, 3}), " ", "_")
		value := Words([]int{1, 5})
		str += fmt.Sprintf("declare -x %s=\"%s\"\n", name, value)
	}

	return str
}
