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
	var c_fds *C.int
	c_cnt := C.size_t(0)

	result := C.launch_activate_socket(c_name, &c_fds, &c_cnt)
	if result != 0 {
		return nil, fmt.Errorf("couldn't activate launchd socket: %s", name)
	}

	pointer := unsafe.Pointer(c_fds)
	defer C.free(pointer)
	length := int(c_cnt)

	listeners := make([]net.Listener, length)
	fds := (*[1 << 30]C.int)(pointer)
	for i := 0; i < length; i++ {
		file := os.NewFile(uintptr(fds[i]), "")
		ln, err := net.FileListener(file)
		if err != nil {
			return nil, fmt.Errorf("couldn't activate launchd socket: %s, fd: %i", name, file.Fd)
		}
		listeners[i] = ln
	}

	return listeners, nil
}
