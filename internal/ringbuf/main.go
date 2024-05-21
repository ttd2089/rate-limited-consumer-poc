package ringbuf

import "errors"

var ErrOutOfRange = errors.New("requested index was out of range")

type Buffer[T any] struct {
	capacity    int
	data        []T
	startOffset int
}

func New[T any](capacity int) Buffer[T] {
	return Buffer[T]{
		capacity:    capacity,
		data:        make([]T, 0, capacity),
		startOffset: 0,
	}
}

func (b *Buffer[T]) Len() int {
	return len(b.data)
}

func (b *Buffer[T]) Push(x T) {
	if len(b.data) < b.capacity {
		b.data = append(b.data, x)
		return
	}
	b.data[b.startOffset] = x
	nextOffset := (b.startOffset + 1)
	nextOffset %= len(b.data)
	b.startOffset = nextOffset
}

func (b *Buffer[T]) Get(i int) (T, error) {
	if i >= len(b.data) {
		var t T
		return t, ErrOutOfRange
	}
	offset := b.startOffset + i
	wrappedOffset := offset % len(b.data)
	return b.data[wrappedOffset], nil
}
