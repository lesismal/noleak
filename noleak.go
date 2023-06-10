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

	// seq
	seq uint64

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
	// a.mark(pbuf)
	uptr := uintptr(unsafe.Pointer(&((*pbuf)[0])))

	// add new buffer to map and set finalizer
	i := a.hash(uptr)
	atomic.AddUint64(&a.nm, 1)
	seq := atomic.AddUint64(&a.seq, 1)
	a.x[i].Lock()
	defer a.x[i].Unlock()
	a.m[i][uptr] = seq
	// runtime.KeepAlive(&buf)
	runtime.SetFinalizer(pbuf, func(p *[]byte) {
		key := uintptr(unsafe.Pointer(&(*p)[0]))
		a.x[i].Lock()
		defer a.x[i].Unlock()
		if a.m[i][key] == seq {
			delete(a.m[i], key)
			a.free(*p)
			atomic.AddUint64(&a.ngf, 1)
		}
	})
	return pbuf
}

func (a *Allocator) Realloc(pbuf *[]byte, size int) *[]byte {
	if size <= cap(*pbuf) {
		*pbuf = (*pbuf)[:size]
		return pbuf
	}
	oldPtr := uintptr(unsafe.Pointer(&(*pbuf)[0]))
	i := a.hash(oldPtr)
	a.x[i].Lock()
	defer a.x[i].Unlock()

	runtime.SetFinalizer(pbuf, nil)
	delete(a.m[i], oldPtr)

	newBuf := a.realloc(*pbuf, size)
	newPtr := uintptr(unsafe.Pointer(&newBuf[0]))
	pNewbuf := &newBuf

	seq := atomic.AddUint64(&a.seq, 1)
	j := a.hash(newPtr)
	if j != i {
		a.x[j].Lock()
		defer a.x[j].Unlock()
	}
	a.m[j][newPtr] = seq
	runtime.SetFinalizer(pNewbuf, func(p *[]byte) {
		key := uintptr(unsafe.Pointer(&(*p)[0]))
		a.x[j].Lock()
		defer a.x[j].Unlock()
		if a.m[j][key] == seq {
			delete(a.m[j], key)
			a.free(*p)
			atomic.AddUint64(&a.ngf, 1)
		}
	})

	if newPtr != oldPtr {
		atomic.AddUint64(&a.nr, 1)
	}

	return pNewbuf
}

func (a *Allocator) Append(buf []byte, more ...byte) *[]byte {
	return a.AppendString(buf, *(*string)(unsafe.Pointer(&more)))
}

func (a *Allocator) AppendString(buf []byte, more string) *[]byte {
	lbuf, lmore := len(buf), len(more)
	pbuf := a.Realloc(&buf, lbuf+lmore)
	copy(buf[lbuf:], more)
	return pbuf
}

func (a *Allocator) Free(pbuf *[]byte) {
	uptr := uintptr(unsafe.Pointer(&((*pbuf)[0])))

	i := a.hash(uptr)
	a.x[i].Lock()
	defer a.x[i].Unlock()

	if _, ok := a.m[i][uptr]; ok {
		atomic.AddUint64(&a.nf, 1)
		delete(a.m[i], uptr)
		runtime.SetFinalizer(pbuf, nil)
		a.free(*pbuf)
	}
}

func (a *Allocator) hash(p uintptr) uint64 {
	u := uint64(p)
	return (u ^ (u >> u)) % a.nb
}

func (a *Allocator) Info() (uint64, uint64, uint64, uint64) {
	return a.nm, a.nr, a.nf, a.ngf
}
