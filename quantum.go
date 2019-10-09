package common

import (
	"fmt"
	"strings"
)

type rang struct {
	from int
	to   int
}

func (r *rang) IsIntersect(o *rang) bool {
	l := r
	u := o

	if l.from > u.from {
		l = o
		u = r
	}

	return l.to >= u.from
}

type Quantum struct {
	ranges []*rang
}

func NewQuantum() *Quantum {
	return &Quantum{make([]*rang, 0)}
}

func isIncluded(min int, max int, v int) bool {
	return min <= v && v <= max
}

func (q *Quantum) IsIncluded(v int) bool {
	return q.findRange(v) != -1
}

func (q *Quantum) findRange(v int) int {
	for i, r := range q.ranges {
		if isIncluded(r.from, r.to, v) {
			return i
		}
	}

	return -1
}

func (q *Quantum) deleteRange(index int) {
	copy(q.ranges[index:], q.ranges[index+1:])
	q.ranges[len(q.ranges)-1] = nil
	q.ranges = q.ranges[:len(q.ranges)-1]
}

func (q *Quantum) insertRange(index int, r *rang) {
	q.ranges = append(q.ranges[:index], append([]*rang{r}, q.ranges[index:]...)...)
}

func (q *Quantum) Get(index int) (int, error) {
	i := index
	for _, r := range q.ranges {
		if isIncluded(0, r.to-r.from, i) {
			return r.from + i, nil
		}
		i -= r.to - r.from + 1
	}

	return 0, fmt.Errorf("index %d out of range %d-%d", index, 0, q.Len())
}

func (q *Quantum) Add(v int) {
	if len(q.ranges) == 0 {
		r := &rang{v, v}
		q.ranges = append(q.ranges, r)

		return
	}

	cmpIndex0 := -1
	cmpIndex1 := -1
	index := 0

	for index < len(q.ranges) {
		r := q.ranges[index]

		if isIncluded(r.from, r.to, v) {
			return
		}

		if v+1 == r.from {
			r.from = v
			if index > 0 {
				cmpIndex0 = index - 1
				cmpIndex1 = index
			}
			break
		}

		if v-1 == r.to {
			r.to = v
			if index < len(q.ranges)-1 {
				cmpIndex0 = index
				cmpIndex1 = index + 1
			}
			break
		}

		index++
	}

	if cmpIndex0 != -1 {
		r0 := q.ranges[cmpIndex0]
		r1 := q.ranges[cmpIndex1]

		if r0.to+1 >= r1.from {
			r0.to = r1.to

			q.deleteRange(cmpIndex1)
		}

		return
	}

	if index == len(q.ranges) {
		q.ranges = append(q.ranges, &rang{v, v})
	}
}

func (q *Quantum) Remove(v int) {
	if len(q.ranges) == 0 {
		return
	}

	index := 0

	for index < len(q.ranges) {
		r := q.ranges[index]

		if isIncluded(r.from, r.to, v) {
			if v == r.from {
				r.from = v + 1

				if r.from > r.to {
					q.deleteRange(index)
				}

				return
			}

			if v == r.to {
				r.to = v - 1

				if r.from > r.to {
					q.deleteRange(index)
				}

				return
			}

			r.to = v - 1

			q.insertRange(index, &rang{v + 1, r.to})

			return
		}

		index++
	}
}

func (q *Quantum) AddRange(from int, to int) {
	if from > to {
		from, to = to, from
	}

	for v := from; v <= to; v++ {
		q.Add(v)
	}
}

func (q *Quantum) RemoveRange(from int, to int) {
	if from > to {
		from, to = to, from
	}

	for v := from; v <= to; v++ {
		q.Remove(v)
	}
}

func (q *Quantum) String() string {
	sb := strings.Builder{}

	for i, r := range q.ranges {
		if i > 0 {
			sb.WriteString(";")
		}
		sb.WriteString(fmt.Sprintf("%d-%d", r.from, r.to))
	}

	return sb.String()
}

func (q *Quantum) Len() int {
	l := 0

	for _, r := range q.ranges {
		l += 1 + r.to - r.from
	}

	return l
}

func (q *Quantum) ToSlice() []int {
	l := make([]int, 0)

	for _, r := range q.ranges {
		for i := r.from; i <= r.to; i++ {
			l = append(l, i)
		}
	}

	return l
}
