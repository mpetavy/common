package common

import "slices"

func SliceRemove[S ~[]E, E comparable](s S, e E) S {
	p := slices.Index(s, e)

	if p != -1 {
		return slices.Delete(s, p, p+1)
	}

	return s
}

func SliceDelete[S ~[]E, E any](s S, index int) S {
	return slices.Delete(s, index, index+1)
}
