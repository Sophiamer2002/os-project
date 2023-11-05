package queue

import "fmt"

type Queue[T any] struct {
	_channel chan T
}

func New[T any]() *Queue[T] {
	var g Queue[T]
	return &g
}

func (self *Queue[T]) Init(capacity int) {
	if capacity <= 0 {
		fmt.Printf("Capacity Error! Setting capacity to 1.")
		capacity = 1
	}
	self._channel = make(chan T, capacity)
}

func (self *Queue[T]) Enqueue(item T) int {
	self._channel <- item
	return 0
}

func (self *Queue[T]) Dequeue() (T, int) {
	front, ok := <-self._channel
	var ret int
	if ok {
		ret = 1
	} else {
		ret = 0
	}
	return front, ret
}

func (self *Queue[T]) Size() int {
	return len(self._channel)
}

func (self *Queue[T]) Capacity() int {
	return cap(self._channel)
}

func (self *Queue[T]) Close() {
	close(self._channel)
}
