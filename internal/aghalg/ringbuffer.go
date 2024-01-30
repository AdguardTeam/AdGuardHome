package aghalg

// RingBuffer is the implementation of ring buffer data structure.
type RingBuffer[T any] struct {
	buf  []T
	cur  uint
	full bool
}

// NewRingBuffer initializes the new instance of ring buffer.  size must be
// greater or equal to zero.
func NewRingBuffer[T any](size uint) (rb *RingBuffer[T]) {
	return &RingBuffer[T]{
		buf: make([]T, size),
	}
}

// Append appends an element to the buffer.
func (rb *RingBuffer[T]) Append(e T) {
	if len(rb.buf) == 0 {
		return
	}

	rb.buf[rb.cur] = e
	rb.cur = (rb.cur + 1) % uint(cap(rb.buf))
	if rb.cur == 0 {
		rb.full = true
	}
}

// Range calls cb for each element of the buffer.  If cb returns false it stops.
func (rb *RingBuffer[T]) Range(cb func(T) (cont bool)) {
	before, after := rb.splitCur()

	for _, e := range before {
		if !cb(e) {
			return
		}
	}

	for _, e := range after {
		if !cb(e) {
			return
		}
	}
}

// ReverseRange calls cb for each element of the buffer in reverse order.  If
// cb returns false it stops.
func (rb *RingBuffer[T]) ReverseRange(cb func(T) (cont bool)) {
	before, after := rb.splitCur()

	for i := len(after) - 1; i >= 0; i-- {
		if !cb(after[i]) {
			return
		}
	}

	for i := len(before) - 1; i >= 0; i-- {
		if !cb(before[i]) {
			return
		}
	}
}

// splitCur splits the buffer in two, before and after current position in
// chronological order.  If buffer is not full, after is nil.
func (rb *RingBuffer[T]) splitCur() (before, after []T) {
	if len(rb.buf) == 0 {
		return nil, nil
	}

	cur := rb.cur
	if !rb.full {
		return rb.buf[:cur], nil
	}

	return rb.buf[cur:], rb.buf[:cur]
}

// Len returns a length of the buffer.
func (rb *RingBuffer[T]) Len() (l uint) {
	if !rb.full {
		return rb.cur
	}

	return uint(cap(rb.buf))
}

// Clear clears the buffer.
func (rb *RingBuffer[T]) Clear() {
	rb.full = false
	rb.cur = 0
}
