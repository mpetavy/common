package common

import "fmt"

// Deque is a double-ended queue.
type Deque[T any] struct {
	head *Node[T]
	tail *Node[T]
	size int
}

// Node represents an element in the deque.
type Node[T any] struct {
	value T
	prev  *Node[T]
	next  *Node[T]
}

// NewDeque creates and returns a new deque.
func NewDeque[T any]() *Deque[T] {
	return &Deque[T]{}
}

// Len returns the current size of the deque.
func (d *Deque[T]) Len() int {
	return d.size
}

// PushFront adds a value to the front of the deque.
func (d *Deque[T]) PushFront(value T) {
	node := &Node[T]{value: value}
	if d.size == 0 {
		d.head = node
		d.tail = node
	} else {
		node.next = d.head
		d.head.prev = node
		d.head = node
	}
	d.size++
}

// PushBack adds a value to the back of the deque.
func (d *Deque[T]) PushBack(value T) {
	node := &Node[T]{value: value}
	if d.size == 0 {
		d.head = node
		d.tail = node
	} else {
		node.prev = d.tail
		d.tail.next = node
		d.tail = node
	}
	d.size++
}

// PopFront removes and returns the value from the front of the deque.
func (d *Deque[T]) PopFront() (T, error) {
	var zero T
	if d.size == 0 {
		return zero, fmt.Errorf("deque is empty")
	}

	value := d.head.value
	d.head = d.head.next
	if d.head != nil {
		d.head.prev = nil
	}
	d.size--

	if d.size == 0 {
		d.tail = nil
	}

	return value, nil
}

// PopBack removes and returns the value from the back of the deque.
func (d *Deque[T]) PopBack() (T, error) {
	var zero T
	if d.size == 0 {
		return zero, fmt.Errorf("deque is empty")
	}

	value := d.tail.value
	d.tail = d.tail.prev
	if d.tail != nil {
		d.tail.next = nil
	}
	d.size--

	if d.size == 0 {
		d.head = nil
	}

	return value, nil
}

// Front returns the value from the front of the deque without removing it.
func (d *Deque[T]) Front() (T, error) {
	var zero T
	if d.size == 0 {
		return zero, fmt.Errorf("deque is empty")
	}
	return d.head.value, nil
}

// Back returns the value from the back of the deque without removing it.
func (d *Deque[T]) Back() (T, error) {
	var zero T
	if d.size == 0 {
		return zero, fmt.Errorf("deque is empty")
	}
	return d.tail.value, nil
}

// Iterate walks through the deque from front to back,
// calling the provided function for each value.
// If the function returns false, iteration stops early.
func (d *Deque[T]) Iterate(fn func(value T) bool) {
	for current := d.head; current != nil; current = current.next {
		if !fn(current.value) {
			break
		}
	}
}
