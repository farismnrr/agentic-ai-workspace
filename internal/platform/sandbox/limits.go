package sandbox

// Limits bound what one sandboxed command may consume. The Docker daemon enforces
// them for the whole container, including every process it starts.
type Limits struct {
	// MemoryBytes caps memory, tmpfs included.
	MemoryBytes uint64
	// SwapBytes is the swap a command may add on top of MemoryBytes; zero keeps swap off. Memory plus swap is
	// still a hard cap for the container, so a runaway command is killed before it can fill the host's swap.
	SwapBytes uint64
	// FileBytes caps the size of any single file the command writes.
	FileBytes uint64
	// Processes caps how many processes and threads a task may have.
	Processes uint64
	// CPUMillis is the CPU share in thousandths of a core.
	CPUMillis uint64
}

const (
	defaultMemory    = 2 << 30
	defaultFileBytes = 512 << 20
	defaultProcesses = 512
	defaultCPUMillis = 1000
)

// DefaultLimits suit builds and test runs of a small project.
func DefaultLimits() Limits {
	return Limits{
		MemoryBytes: defaultMemory, FileBytes: defaultFileBytes, Processes: defaultProcesses,
		CPUMillis: defaultCPUMillis,
	}
}
