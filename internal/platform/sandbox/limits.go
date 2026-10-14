package sandbox

// Limits bound what one sandboxed command may consume. Memory, CPU and process
// counts are enforced by a cgroup when the server has one (see OpenCgroups);
// otherwise only the rlimits below apply, which are coarser: they work without
// root but cannot account for a whole task.
type Limits struct {
	// MemoryBytes caps memory. With a cgroup it is exact and includes page cache and
	// tmpfs; as an rlimit (RLIMIT_DATA) it only covers private writable memory.
	MemoryBytes uint64
	// FileBytes caps the size of any single file the command writes.
	FileBytes uint64
	// Processes caps how many processes and threads a task may have (cgroup only).
	Processes uint64
	// CPUMillis is the CPU share in thousandths of a core (cgroup only).
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
