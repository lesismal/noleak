package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/lesismal/noleak"
)

func main() {
	allocator := noleak.New()

	for i := 0; i < 1000; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				b := allocator.Malloc(j + 1)
				if j%2 == 0 {
					allocator.Free(b)
				}
			}
		}()
	}

	for {
		time.Sleep(time.Second)
		nm, nf, ngf := allocator.Info()
		fmt.Println("-------------------------------")
		log.Printf("malloc : %v", nm)
		log.Printf("free   : %v", nf)
		log.Printf("gc free: %v", nf)
		log.Printf("to free: %v", nm-nf-ngf)
		runtime.GC()
	}
}
