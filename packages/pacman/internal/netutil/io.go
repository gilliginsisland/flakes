package netutil

import (
	"io"
	"sync"
)

func Pipe(a, b io.ReadWriter) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(a, b)
	}()

	go func() {
		defer wg.Done()
		io.Copy(b, a)
	}()

	wg.Wait()
}
