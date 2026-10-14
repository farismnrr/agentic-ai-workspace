package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
)

// failureStatus is the exit code the shell uses for "found but could not run".
const failureStatus = 126

// initArgs are the arguments after the init flag: new root, toolbox, workspace,
// three limits, a "--" separator, then the command.
const initArgs = 7

// RunIfChild turns this process into the sandbox init when it was started by
// Command. It must be the first thing main does and never returns in that case.
func RunIfChild() {
	if len(os.Args) >= 2 && os.Args[1] == buildFlag {
		runBuild(os.Args[2:])
	}

	if len(os.Args) < 2 || os.Args[1] != initFlag {
		return
	}

	// Capability and no-new-privileges settings are per thread, and so is the
	// exec below: stay on one thread for the whole setup.
	runtime.LockOSThread()

	if err := enter(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "could not start the command:", err)
		os.Exit(failureStatus)
	}
}

func enter(args []string) error {
	if len(args) <= initArgs || args[initArgs-1] != "--" {
		return errBadArguments
	}

	limits, err := parseLimits(args[3:6])
	if err != nil {
		return err
	}

	if err = buildRoot(args[0], args[1], args[2]); err != nil {
		return err
	}

	if err = harden(limits); err != nil {
		return err
	}

	command := args[initArgs:]

	path, err := exec.LookPath(command[0])
	if err != nil {
		return fmt.Errorf("find command: %w", err)
	}

	//nolint:gosec // the command is the approved task this sandbox exists to run
	if err = syscall.Exec(path, command, os.Environ()); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	return nil
}

func parseLimits(values []string) (Limits, error) {
	var numbers [3]uint64

	for i, value := range values {
		parsed, err := strconv.ParseUint(value, base, 64)
		if err != nil {
			return Limits{}, fmt.Errorf("%w: %w", errBadArguments, err)
		}

		numbers[i] = parsed
	}

	return Limits{MemoryBytes: numbers[0], FileBytes: numbers[1], Processes: numbers[2]}, nil
}
