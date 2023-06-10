package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/lesismal/noleak"
)

func main() {
	N := 300
	size := 1024
	allocator := noleak.New(1)

	buffers := make([][]*[]byte, N)

	for i := 0; i < N; i++ {
		buffers[i] = make([]*[]byte, N)
		go func(idx int) {
			for j := 0; j < N; j++ {
				pbuf0 := allocator.Malloc(size)
				pbuf := allocator.Realloc(pbuf0, size*2)
				buffers[idx][j] = pbuf
				if j%2 == 1 {
					allocator.Free(pbuf)
				} else {
					go func(p *[]byte) {
						for x := 0; x < 3; x++ {
							time.Sleep(time.Second)
							for k := 0; k < size; k++ {
								(*p)[k] = (*p)[k] + 1
							}
						}
					}(pbuf)
				}
			}
		}(i)
	}

	for i := 1; i <= 5; i++ {
		time.Sleep(time.Second * 1)
		nm, nr, nf, ngf := allocator.Info()
		fmt.Println("-------------------------------\n" +
			fmt.Sprintf("count  : %v\n", i) +
			fmt.Sprintf("malloc : %v\n", nm) +
			fmt.Sprintf("realloc: %v\n", nr) +
			fmt.Sprintf("free   : %v\n", nf) +
			fmt.Sprintf("gc free: %v\n", ngf) +
			fmt.Sprintf("to free: %v\n", nm-nf-ngf))

		runtime.GC()
	}
}
