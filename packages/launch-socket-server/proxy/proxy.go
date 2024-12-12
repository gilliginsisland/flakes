package proxy

import (
	"fmt"
	"io"
	"net"
)

func New(dst net.Addr) *Proxy {
	return &Proxy{dst}
}

type Proxy struct {
	dst net.Addr
}

func (p *Proxy) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("error accepting connection: %w", err)
		}

		go p.Handle(conn)
	}
}

func (p *Proxy) Handle(in net.Conn) error {
	defer in.Close()

	out, err := p.dial()
	if err != nil {
		return fmt.Errorf("error connecting to dst '%s' : %w", p.dst.String(), err)
	}
	defer out.Close()

	done := make(chan error, 2)
	pipe := func(w io.WriteCloser, r io.ReadCloser) {
		_, err := io.Copy(w, r)
		done <- err
	}
	go pipe(in, out)
	go pipe(out, in)

	if err := <-done; err != nil {
		return err
	}

	return nil
}

func (p *Proxy) Reachable() bool {
	out, err := p.dial()
	if out != nil {
		out.Close()
	}
	return err == nil
}

func (p *Proxy) dial() (net.Conn, error) {
	return net.Dial(p.dst.Network(), p.dst.String())
}
