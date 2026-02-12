package launch

/*
#include <stdlib.h>
int launch_activate_socket(const char *name, int **fds, size_t *cnt);
*/
import "C"

import (
	"fmt"
	"net"
	"os"
	"unsafe"
)

func ActivateSocket(name string) ([]net.Listener, error) {
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))

	var (
		c_fds *C.int
		c_cnt C.size_t
	)

	result := C.launch_activate_socket(c_name, &c_fds, &c_cnt)
	if result != 0 {
		return nil, fmt.Errorf("error activating launch socket: %s", name)
	}
	defer C.free(unsafe.Pointer(c_fds))

	fds := unsafe.Slice(c_fds, int(c_cnt))

	listeners := make([]net.Listener, len(fds))
	for i, fd := range fds {
		file := os.NewFile(uintptr(fd), "")
		ln, err := net.FileListener(file)
		if err != nil {
			return nil, fmt.Errorf("error activating launch socket: %s, fd: %d", name, file.Fd())
		}
		listeners[i] = ln
	}

	return listeners, nil
}
