package runner_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// A vendor's coding program is given its prompt on standard input and keeps its sign-in in the home folder.
func TestLeaseGivesTheProgramItsInputAndHome(t *testing.T) {
	t.Parallel()
	requireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	home := t.TempDir()

	docker, spec, release, err := engine.Lease(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	defer release()

	spec.Home, spec.Stdin = home, strings.NewReader("hello from the prompt\n")
	spec.Argv = []string{"sh", "-c", "cat > /home/cli/in.txt && cat /home/cli/in.txt"}

	var out bytes.Buffer

	result, err := docker.Run(ctx, spec, &out)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("run: %v, %+v, output %q", err, result, out.String())
	}

	folder, err := os.OpenRoot(home)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = folder.Close() }()

	kept, err := folder.ReadFile("in.txt")
	if err != nil || string(kept) != "hello from the prompt\n" || out.String() != string(kept) {
		t.Fatalf("home file %q (%v), output %q", kept, err, out.String())
	}
}
