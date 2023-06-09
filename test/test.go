package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/lesismal/noleak"
)

func main() {
	N := 100
	size := 1024
	allocator := noleak.New(1)

	buffers := make([][]*[]byte, N)

	for i := 0; i < N; i++ {
		buffers[i] = make([]*[]byte, N)
		go func(idx int) {
			for j := 0; j < N; j++ {
				pbuf := allocator.Malloc(size)
				buffers[idx][j] = pbuf
				// pbuf = allocator.Realloc(*pbuf, size*2)
				if j%2 == 0 {
					allocator.Free(*pbuf)
				} else {
					go func(pbuf *[]byte) {
						for x := 0; x < 5; x++ {
							time.Sleep(time.Second)
							for k := 0; k < size; k++ {
								(*pbuf)[k] = (*pbuf)[k] + 1
							}
						}
					}(pbuf)
				}
			}
		}(i)
	}

	for {
		time.Sleep(time.Second * 1)
		nm, nr, nf, ngf := allocator.Info()
		fmt.Println("-------------------------------")
		log.Printf("malloc : %v", nm)
		log.Printf("realloc: %v", nr)
		log.Printf("free   : %v", nf)
		log.Printf("gc free: %v", ngf)
		log.Printf("to free: %v", nm-nf-ngf)

		runtime.GC()

		// if nm-nf-ngf == 0 {
		// 	for i := 0; i < N; i++ {
		// 		for j := 0; j < N; j++ {
		// 			for k := 0; k < j+1; k++ {
		// 				buffers[i][j][k] = byte(k)
		// 			}
		// 		}
		// 	}
		// }
	}
}
