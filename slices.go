package common

func SliceIndex[S ~[]E, E comparable](s S, e E) int {
	for i, t := range s {
		if t == e {
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

func SliceInsert[S ~[]E, E comparable](s S, index int, e ...E) S {
	n := make([]E, len(s)+len(e))

	copy(n, s[:index])
	copy(n[index:], e)
	copy(n[index+len(e):], s[index:])

	return n
}

func SliceDeleteLen[S ~[]E, E any](s S, index int, length int) S {
	n := make([]E, len(s)-length)

	copy(n, s[:index])
	copy(n[index:], s[index+length:])

	return n
}

func SliceDelete[S ~[]E, E any](s S, index int) S {
	return SliceDeleteLen(s, index, 1)
}
