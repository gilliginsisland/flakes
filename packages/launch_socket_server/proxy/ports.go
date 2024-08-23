package proxy

import (
	"fmt"
	"net"
)

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort(ip net.IP) (int, error) {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   ip,
		Port: 0,
	})
	if err != nil {
		return 0, err
	}
	defer l.Close()

	laddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("listen address doesn't match requested address")
	}

	return laddr.Port, nil
}
