package netutil

import (
	"io"
	"net"
	"sync"
)

func Pipe(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	copy := func(dst, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		io.Copy(dst, src)
	}

	go copy(b, a)
	go copy(a, b)

	wg.Wait()
}
