package tll

import "C"
import "syscall"

type Errno = syscall.Errno

var (
	eEAGAIN error = syscall.EAGAIN
	eEINVAL error = syscall.EINVAL
	eENOSYS error = syscall.ENOSYS
)

// Poor man's syscall.errnoErr function
func cint2error(e C.int) error {
	switch e {
	case 0: return nil
	case C.int(syscall.EAGAIN): return eEAGAIN
	case C.int(syscall.EINVAL): return eEINVAL
	case C.int(syscall.ENOSYS): return eENOSYS
	}
	return Errno(int(e))
}
