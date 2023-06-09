package noleak

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

const defaultAllocatiroThreadNumLimit uint64 = 1

type Allocator struct {
	// bucketNum
	nb uint64

	// mux buckets
	x []sync.Mutex
	// map buckets
	m []map[uintptr]uint64

	// malloc times
	nm uint64
	// realloc times
	nr uint64
	// free times by user
	nf uint64
	// free times by gc finalizer
	ngf uint64

	malloc  func(size int) []byte
	realloc func(buf []byte, size int) []byte
	free    func(buf []byte)
}

func New(maxThreadNum uint64) *Allocator {
	if maxThreadNum == 0 {
		maxThreadNum = defaultAllocatiroThreadNumLimit
	}
	a := &Allocator{
		nb:      maxThreadNum,
		x:       make([]sync.Mutex, maxThreadNum),
		m:       make([]map[uintptr]uint64, maxThreadNum),
		malloc:  GLIBC_Malloc,
		realloc: GLIBC_Realloc,
		free:    GLIBC_Free,
	}
	for i := uint64(0); i < maxThreadNum; i++ {
		a.m[i] = map[uintptr]uint64{}
	}
	return a
}

func (a *Allocator) Malloc(size int) *[]byte {
	buf := a.malloc(size)
	pbuf := &buf
	a.mark(pbuf)
	return pbuf
}

func (a *Allocator) mark(pbuf *[]byte) {
	uptr := uintptr(unsafe.Pointer(&((*pbuf)[0])))

	// add new buffer to map and set finalizer
	i := a.hash(uptr)
	nm := atomic.AddUint64(&a.nm, 1)
	a.x[i].Lock()
	defer a.x[i].Unlock()
	a.m[i][uptr] = nm
	// runtime.KeepAlive(&buf)
	runtime.SetFinalizer(pbuf, func(p *[]byte) {
		key := uintptr(unsafe.Pointer(&(*p)[0]))
		a.x[i].Lock()
		defer a.x[i].Unlock()
		if a.m[i][key] == nm {
			delete(a.m[i], key)
			a.free(*p)
			atomic.AddUint64(&a.ngf, 1)
		}
	})
}

func (a *Allocator) unmark(pbuf *[]byte) {
	uptr := uintptr(unsafe.Pointer(&((*pbuf)[0])))
	i := a.hash(uptr)
	a.x[i].Lock()
	defer a.x[i].Unlock()
	delete(a.m[i], uptr)

	runtime.SetFinalizer(pbuf, nil)
}

func (a *Allocator) Realloc(buf []byte, size int) *[]byte {
	if size <= cap(buf) {
		buf = buf[:size]
		return &buf
	}
	oldPtr := uintptr(unsafe.Pointer(&buf[0]))
	i := a.hash(oldPtr)
	a.x[i].Lock()
	newBuf := a.realloc(buf, size)
	a.x[i].Unlock()

	newPtr := uintptr(unsafe.Pointer(&newBuf[0]))
	if newPtr != oldPtr {
		a.unmark(&buf)
		a.mark(&newBuf)
		atomic.AddUint64(&a.nr, 1)
	}

	return &newBuf
}

func (a *Allocator) Append(buf []byte, more ...byte) *[]byte {
	return a.AppendString(buf, *(*string)(unsafe.Pointer(&more)))
}

func (a *Allocator) AppendString(buf []byte, more string) *[]byte {
	lbuf, lmore := len(buf), len(more)
	pbuf := a.Realloc(buf, lbuf+lmore)
	copy((*pbuf)[lbuf:], more)
	return pbuf
}

func (a *Allocator) Free(buf []byte) {
	uptr := uintptr(unsafe.Pointer(&buf[0]))

	i := a.hash(uptr)
	a.x[i].Lock()
	defer a.x[i].Unlock()

	if _, ok := a.m[i][uptr]; ok {
		atomic.AddUint64(&a.nf, 1)
		delete(a.m[i], uptr)
		runtime.SetFinalizer(&buf, nil)
		a.free(buf)
	}
}

func (a *Allocator) hash(p uintptr) uint64 {
	u := uint64(p)
	return (u ^ (u >> u)) % a.nb
}

func (a *Allocator) Info() (uint64, uint64, uint64, uint64) {
	return a.nm, a.nr, a.nf, a.ngf
}
