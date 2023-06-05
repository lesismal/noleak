package noleak

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const bucketNum = 512

type Allocator struct {
	// mux buckets
	x [bucketNum]sync.Mutex
	// map buckets
	m [bucketNum]map[uintptr]uint64

	// malloc times
	nm uint64
	// realloc times
	nr uint64
	// free times by user
	nf uint64
	// free times by gc finalizer
	ngf uint64
}

func (a *Allocator) Malloc(size int) []byte {
	b := make([]byte, size)
	uptr := uintptr(unsafe.Pointer(&b[0]))
	nm := atomic.AddUint64(&a.nm, 1)

	i := hash(uptr)
	a.x[i].Lock()
	defer a.x[i].Unlock()

	a.m[i][uptr] = nm

	runtime.SetFinalizer(&b, func(p *[]byte) {
		key := uintptr(unsafe.Pointer(&(*p)[0]))
		a.x[i].Lock()
		defer a.x[i].Unlock()
		if a.m[i][key] == nm {
			delete(a.m[i], key)
			// free(buf)

			// debug
			atomic.AddUint64(&a.ngf, 1)
		}

	})
	return b
}

func (a *Allocator) Realloc(buf []byte, size int) []byte {
	if size <= cap(buf) {
		return buf[:size]
	}
	uptr := uintptr(unsafe.Pointer(&buf[0]))

	i := hash(uptr)
	a.x[i].Lock()
	defer a.x[i].Unlock()

	return append(buf[:], make([]byte, size-cap(buf))...)
}

func (a *Allocator) Append(buf []byte, more ...byte) []byte {
	return a.AppendString(buf, *(*string)(unsafe.Pointer(&more)))
}

func (a *Allocator) AppendString(buf []byte, more string) []byte {
	lbuf, lmore := len(buf), len(more)
	buf = a.Realloc(buf, lbuf+lmore)
	copy(buf[lbuf:], more)
	return buf
}

func (a *Allocator) Free(buf []byte) {
	uptr := uintptr(unsafe.Pointer(&buf[0]))

	i := hash(uptr)
	a.x[i].Lock()
	defer a.x[i].Unlock()

	// debug
	if _, ok := a.m[i][uptr]; ok {
		atomic.AddUint64(&a.nf, 1)
		delete(a.m[i], uptr)

		runtime.SetFinalizer(&buf, nil)
		// free(buf)
	}

	// c_free()
}

func (a *Allocator) Info() (uint64, uint64, uint64) {
	return a.nm, a.nf, a.ngf
}

func New() *Allocator {
	a := &Allocator{}
	for i := range a.m {
		a.m[i] = map[uintptr]uint64{}
	}
	return a
}

func hash(p uintptr) uint64 {
	u := uint64(p)
	return (u ^ (u >> u)) % bucketNum
}
