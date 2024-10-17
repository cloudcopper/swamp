package lib

func First[T any](first T, rest ...interface{}) T {
	return first
}
