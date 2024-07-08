package write

type Write[T any] interface {
	Write(T) ([]any, error)
}
