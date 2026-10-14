//go:build linux && (amd64 || arm64)

package sandbox

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Offsets into the kernel's seccomp_data record: the call number, the CPU
// architecture, and the first argument (little-endian on both supported CPUs).
const (
	offsetNumber = 0
	offsetArch   = 4
	offsetArg0   = 16
)

// denied are system calls an ordinary build or test never needs and that widen the
// kernel attack surface or reach host-wide state: mounting, new namespaces, the
// kernel keyring, BPF, user-space page faults, io_uring, tracing, module loading,
// clocks, swap and a few more. The list follows Docker's default profile; it is a
// deny list, so anything not named stays allowed.
var denied = []uint32{
	unix.SYS_MOUNT, unix.SYS_UMOUNT2, unix.SYS_PIVOT_ROOT, unix.SYS_CHROOT, unix.SYS_UNSHARE, unix.SYS_SETNS,
	unix.SYS_MOVE_MOUNT, unix.SYS_OPEN_TREE, unix.SYS_FSOPEN, unix.SYS_FSCONFIG, unix.SYS_FSMOUNT,
	unix.SYS_FSPICK, unix.SYS_MOUNT_SETATTR,
	unix.SYS_BPF, unix.SYS_USERFAULTFD, unix.SYS_PERF_EVENT_OPEN, unix.SYS_LOOKUP_DCOOKIE,
	unix.SYS_IO_URING_SETUP, unix.SYS_IO_URING_ENTER, unix.SYS_IO_URING_REGISTER,
	unix.SYS_KEYCTL, unix.SYS_ADD_KEY, unix.SYS_REQUEST_KEY,
	unix.SYS_KEXEC_LOAD, unix.SYS_KEXEC_FILE_LOAD, unix.SYS_INIT_MODULE, unix.SYS_FINIT_MODULE,
	unix.SYS_DELETE_MODULE, unix.SYS_REBOOT,
	unix.SYS_SETTIMEOFDAY, unix.SYS_CLOCK_SETTIME, unix.SYS_CLOCK_ADJTIME, unix.SYS_ADJTIMEX,
	unix.SYS_ACCT, unix.SYS_QUOTACTL, unix.SYS_SWAPON, unix.SYS_SWAPOFF, unix.SYS_SYSLOG,
	unix.SYS_OPEN_BY_HANDLE_AT, unix.SYS_NAME_TO_HANDLE_AT,
	unix.SYS_PROCESS_VM_READV, unix.SYS_PROCESS_VM_WRITEV, unix.SYS_KCMP, unix.SYS_PIDFD_GETFD,
	unix.SYS_MBIND, unix.SYS_MOVE_PAGES, unix.SYS_SET_MEMPOLICY,
}

// Errors a refused call reports. A refusal looks like missing permission, except
// clone3, which looks like an unknown call so libraries fall back to clone.
const (
	errPermission  = uint32(unix.EPERM)
	errUnknownCall = uint32(unix.ENOSYS)
)

// Lengths of the rule bodies below, which a failed match must jump over.
const (
	cloneBodyLength  = 4
	socketBodyLength = 6
)

// namespaceFlags are the clone flags that create a namespace.
const namespaceFlags = unix.CLONE_NEWNS | unix.CLONE_NEWCGROUP | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC |
	unix.CLONE_NEWUSER | unix.CLONE_NEWPID | unix.CLONE_NEWNET

// Socket families that expose kernel features meant for privileged users.
const (
	familyAlg   = 38
	familyVsock = 40
)

func load(offset uint32) unix.SockFilter {
	return unix.SockFilter{Code: unix.BPF_LD | unix.BPF_W | unix.BPF_ABS, K: offset}
}

func jump(code uint16, value uint32, onTrue, onFalse uint8) unix.SockFilter {
	return unix.SockFilter{Code: unix.BPF_JMP | code | unix.BPF_K, K: value, Jt: onTrue, Jf: onFalse}
}

func result(value uint32) unix.SockFilter {
	return unix.SockFilter{Code: unix.BPF_RET | unix.BPF_K, K: value}
}

func refuse(errno uint32) unix.SockFilter {
	return result(unix.SECCOMP_RET_ERRNO | errno)
}

// buildFilter assembles the program. Each rule reloads the call number when it
// had to look at an argument, so the rules that follow still see it.
func buildFilter() []unix.SockFilter {
	program := []unix.SockFilter{
		// A call made through another ABI (32-bit on x86-64) would dodge the numbers below.
		load(offsetArch), jump(unix.BPF_JEQ, auditArch, 1, 0), refuse(errPermission),
		load(offsetNumber),
	}

	if x32Bit != 0 {
		program = append(program, jump(unix.BPF_JGE, x32Bit, 0, 1), refuse(errPermission))
	}

	for _, number := range denied {
		program = append(program, jump(unix.BPF_JEQ, number, 0, 1), refuse(errPermission))
	}

	program = append(program,
		// clone may not create namespaces. clone3 hides its flags behind a pointer, so it
		// is refused as unknown and libraries fall back to clone.
		jump(unix.BPF_JEQ, unix.SYS_CLONE, 0, cloneBodyLength), load(offsetArg0),
		jump(unix.BPF_JSET, namespaceFlags, 0, 1), refuse(errPermission), load(offsetNumber),
		jump(unix.BPF_JEQ, unix.SYS_CLONE3, 0, 1), refuse(errUnknownCall),
		jump(unix.BPF_JEQ, unix.SYS_SOCKET, 0, socketBodyLength), load(offsetArg0),
		jump(unix.BPF_JEQ, familyAlg, 0, 1), refuse(errPermission),
		jump(unix.BPF_JEQ, familyVsock, 0, 1), refuse(errPermission),
		load(offsetNumber),
		result(unix.SECCOMP_RET_ALLOW),
	)

	return program
}

// installFilter applies the filter to this thread and everything it execs. The
// kernel requires no-new-privileges to be set first, which harden has done.
func installFilter() error {
	program := buildFilter()
	//nolint:gosec // the program is a few dozen instructions, far below the 65535 limit
	filter := unix.SockFprog{Len: uint16(len(program)), Filter: &program[0]}

	//nolint:gosec // the kernel interface takes a pointer to the program
	_, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, 0, uintptr(unsafe.Pointer(&filter)))
	runtime.KeepAlive(program)

	if errno != 0 {
		return fmt.Errorf("install system call filter: %w", errno)
	}

	return nil
}
