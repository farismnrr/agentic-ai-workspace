package sandbox

// auditArch names the 64-bit x86 ABI in a seccomp record; x32Bit marks the x32 ABI's call numbers.
const (
	auditArch = 0xC000003E
	x32Bit    = 0x40000000
)
