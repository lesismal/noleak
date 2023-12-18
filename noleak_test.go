package noleak

import (
	"sync"
	"testing"
)

const (
	bufSize     = 1024
	concurrency = 100
	allocTimes  = 1000
)

func poolGetPutTask(pool *sync.Pool) {
	ch := make(chan *[]byte, allocTimes)
	for i := 0; i < concurrency; i++ {
		go func(n int) {
			for j := 0; j < allocTimes; j++ {
				p := pool.Get().(*[]byte)
				ch <- p
			}
		}(i)
	}
	for i := 0; i < allocTimes*concurrency; i++ {
		p := <-ch
		pool.Put(p)
	}
}

func poolAppendTask(pool *sync.Pool) {
	ch := make(chan *[]byte, allocTimes)
	for i := 0; i < concurrency; i++ {
		go func(n int) {
			for j := 0; j < allocTimes; j++ {
				p := pool.Get().(*[]byte)
				*p = (*p)[:bufSize]
				*p = append(*p, make([]byte, bufSize)...)
				ch <- p
			}
		}(i)
	}
	for i := 0; i < allocTimes*concurrency; i++ {
		p := <-ch
		pool.Put(p)
	}
}

func mallocTask() {
	ch := make(chan *[]byte, allocTimes)
	for i := 0; i < concurrency; i++ {
		go func(n int) {
			for j := 0; j < allocTimes; j++ {
				p := Malloc(bufSize)
				ch <- p
			}
		}(i)
	}
	for i := 0; i < allocTimes*concurrency; i++ {
		p := <-ch
		Free(p)
	}
}

func reallocTask() {
	ch := make(chan *[]byte, allocTimes)
	for i := 0; i < concurrency; i++ {
		go func(n int) {
			for j := 0; j < allocTimes; j++ {
				p := Malloc(bufSize)
				p = Realloc(p, bufSize)
				ch <- p
			}
		}(i)
	}
	for i := 0; i < allocTimes*concurrency; i++ {
		p := <-ch
		Free(p)
	}
}

func Benchmark_SyncPool_GetPut(b *testing.B) {
	pool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, bufSize)
			return &b
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poolGetPutTask(pool)
	}
}

func Benchmark_GLIBC_MallocFree(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mallocTask()
	}
}

func Benchmark_SyncPool_Append(b *testing.B) {
	pool := &sync.Pool{
		New: func() interface{} {
			b := make([]byte, bufSize)
			return &b
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		poolAppendTask(pool)
	}
}

func Benchmark_GLIBC_ReallocFree(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reallocTask()
	}
}

