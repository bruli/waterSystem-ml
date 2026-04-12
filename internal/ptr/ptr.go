package ptr

func ToPointer[T any](v T) *T {
	return &v
}

func FromPointer[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
