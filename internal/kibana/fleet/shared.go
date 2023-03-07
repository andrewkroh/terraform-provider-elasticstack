package fleet

func ptrTo[T any](in T) *T {
	return &in
}
