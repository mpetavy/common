package common

func SliceClone[S ~[]E, E any](s S) S {
	n := make([]E, len(s))

	copy(n, s)

	return n
}

func SliceContains[S ~[]E, E comparable](s S, e E) bool {
	return SliceIndex(s, e) != -1
}

func SliceIndex[S ~[]E, E comparable](s S, e E) int {
	for i, t := range s {
		if t == e {
			return i
		}
	}

	return -1
}

func SliceIndexFunc[S ~[]E, E any](s S, fn func(E) bool) int {
	for i, t := range s {
		if fn(t) {
			return i
		}
	}

	return -1
}

func SliceAppend[S ~[]E, E any](s S, e ...E) S {
	return append(s, e...)
}

func SliceRemove[S ~[]E, E comparable](s S, e E) S {
	p := SliceIndex(s, e)

	if p == -1 {
		return s
	}

	n := make([]E, len(s)-1)

	copy(n, s[:p])
	copy(n[p:], s[p+1:])

	return n
}

func SliceInsert[S ~[]E, E any](s S, index int, e ...E) S {
	n := make([]E, len(s)+len(e))

	copy(n, s[:index])
	copy(n[index:], e)
	copy(n[index+len(e):], s[index:])

	return n
}

func SliceDeleteRange[S ~[]E, E any](s S, index0 int, index1 int) S {
	n := make([]E, len(s)-(index1-index0))

	copy(n, s[:index0])
	copy(n[index0:], s[index1:])

	return n
}

func SliceDelete[S ~[]E, E any](s S, index int) S {
	return SliceDeleteRange(s, index, index+1)
}
