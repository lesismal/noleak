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

	// n malloc
	nm uint64
	// n free
	nf uint64
}

func (a *Allocator) Malloc(size int) []byte {
	b := make([]byte, size)
	uptr := uintptr(unsafe.Pointer(&b[0]))
	nm := atomic.AddUint64(&a.nm, 1)

	i := hash(uptr)
	a.x[i].Lock()
	a.m[i][uptr] = nm
	a.x[i].Unlock()

	runtime.SetFinalizer(&b, func(p *[]byte) {
		key := uintptr(unsafe.Pointer(&(*p)[0]))
		a.x[i].Lock()
		if a.m[i][key] == nm {
			// c_free()

			delete(a.m[i], key)
			// debug
			atomic.AddUint64(&a.nf, 1)
		}
		a.x[i].Unlock()
	})
	return b
}

func (a *Allocator) Free(buf []byte) {
	uptr := uintptr(unsafe.Pointer(&buf[0]))

	i := hash(uptr)
	a.x[i].Lock()
	delete(a.m[i], uptr)

	// debug
	if _, ok := a.m[i][uptr]; ok {
		atomic.AddUint64(&a.nf, 1)
	}
	a.x[i].Unlock()

	// c_free()
}

func (a *Allocator) Info() {

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
