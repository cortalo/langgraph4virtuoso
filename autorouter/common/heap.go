package common

import "container/heap"

type MinHeap[T any] struct {
	items []T
	less  func(a, b T) bool
}

func NewMinHeap[T any](less func(a, b T) bool) *MinHeap[T] {
	h := &MinHeap[T]{less: less}
	heap.Init(h)
	return h
}

func (h *MinHeap[T]) Len() int           { return len(h.items) }
func (h *MinHeap[T]) Less(i, j int) bool { return h.less(h.items[i], h.items[j]) }
func (h *MinHeap[T]) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *MinHeap[T]) Push(x interface{}) { h.items = append(h.items, x.(T)) }
func (h *MinHeap[T]) Pop() interface{} {
	old := h.items
	n := old[len(old)-1]
	h.items = old[:len(old)-1]
	return n
}

func (h *MinHeap[T]) PushItem(item T) { heap.Push(h, item) }
func (h *MinHeap[T]) PopItem() T      { return heap.Pop(h).(T) }
