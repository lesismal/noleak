package noleak

/*
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

type _slice struct {
	array unsafe.Pointer
	len   int
	cap   int
}

func GLIBC_Malloc(size int) []byte {
	var buf []byte
	ptr := &buf
	pbuf := C.calloc(1, C.size_t(size))
	slice := _slice{
		array: unsafe.Pointer(pbuf),
		len:   size,
		cap:   size,
	}
	*((*_slice)(unsafe.Pointer(ptr))) = slice
	// log.Println("malloc :", &buf[0])
	return buf
}

func GLIBC_Realloc(buf []byte, size int) []byte {
	pslice := (*_slice)(unsafe.Pointer(&buf))
	pbuf := C.realloc(pslice.array, C.size_t(size))
	pslice.array = unsafe.Pointer(pbuf)
	pslice.len = size
	pslice.cap = size
	// log.Println("realloc:", &buf[0])
	return buf
}

func GLIBC_Free(buf []byte) {
	// log.Println("free   :", &buf[0])
	slice := _slice{}
	pslice := &slice
	*pslice = *((*_slice)(unsafe.Pointer(&buf)))
	C.free(pslice.array)
}
