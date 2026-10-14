package sandbox

// auditArch names the 64-bit ARM ABI in a seccomp record; arm64 has no x32 ABI.
const (
	auditArch = 0xC00000B7
	x32Bit    = 0
)
