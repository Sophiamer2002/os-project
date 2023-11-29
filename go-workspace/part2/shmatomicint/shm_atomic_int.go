package shmatomicint

// #cgo CFLAGS: -std=c11
// #cgo LDFLAGS: -lrt -lpthread
// #include "shm_atomic_int.h"
// #include <stdlib.h>
import "C"

import "unsafe"

type ShmAtomicInt struct {
	ptr  unsafe.Pointer
	name string
}

func New(name string, initial int) (*ShmAtomicInt, error) {
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))

	ptr, err := C.shmint_new(c_name, C.int(initial))
	if err != nil {
		return nil, err
	}

	return &ShmAtomicInt{ptr: ptr, name: name}, nil
}

func Bind(name string) (*ShmAtomicInt, error) {
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))

	ptr, err := C.shmint_bind(c_name)
	if err != nil {
		return nil, err
	}

	return &ShmAtomicInt{ptr: ptr, name: name}, nil
}

func (s *ShmAtomicInt) Unlink() error {
	c_name := C.CString(s.name)
	defer C.free(unsafe.Pointer(c_name))

	_, err := C.shmint_unlink(c_name)
	return err
}

func (s *ShmAtomicInt) AtomicStore(val int) {
	C.shmint_atomic_store(s.ptr, C.int(val))
}

func (s *ShmAtomicInt) AtomicLoad() int {
	return int(C.shmint_atomic_load(s.ptr))
}

func (s *ShmAtomicInt) AtomicExchange(val int) int {
	return int(C.shmint_atomic_exchange(s.ptr, C.int(val)))
}

func (s *ShmAtomicInt) AtomicCompareExchange(val, expected int) int {
	return int(C.shmint_atomic_compare_exchange(s.ptr, C.int(val), C.int(expected)))
}

func (s *ShmAtomicInt) AtomicFetchAdd(val int) int {
	return int(C.shmint_atomic_fetch_add(s.ptr, C.int(val)))
}

func (s *ShmAtomicInt) AtomicFetchSub(val int) int {
	return int(C.shmint_atomic_fetch_sub(s.ptr, C.int(val)))
}
