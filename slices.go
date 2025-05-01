package common

import "slices"

func SliceRemove[S ~[]E, E comparable](s S, e ...E) S {
	for i := 0; i < len(e); i++ {
		p := slices.Index(s, e[i])

		if p != -1 {
			return slices.Delete(s, p, p+1)
		}
	}

	return s
}

func SliceDelete[S ~[]E, E any](s S, index int) S {
	return slices.Delete(s, index, index+1)
}

func SliceMove[S ~[]E, E any](s S, from int, to int) S {
	if from == to {
		return s
	}

	// Remove the element from the original position
	e := s[from]
	s = append(s[:from], s[from+1:]...)

	// Insert the element at the new position
	if to == 0 {
		s = append([]E{e}, s...)
	} else if to == len(s) {
		s = append(s, e)
	} else {
		s = append(s[:to], append([]E{e}, s[to:]...)...)
	}

	return s
}
