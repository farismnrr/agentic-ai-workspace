package domain

import (
	"errors"
	"strings"
)

// ErrBadVerifyArgs keeps QA's run_qa command bound to common verification
// entrypoints. The sandbox (read-only workspace, no credentials/egress) is the
// actual isolation boundary: command naming alone is not a security sandbox.
var ErrBadVerifyArgs = errors.New("verification command must be a supported test, lint or check executable")

// VerifyTask allows tests and linters without requiring a .workspace/tasks.json
// entry. Each command executes argv directly, without bash or project writes.
func VerifyTask(args []string) (Task, error) {
	if len(args) < 2 || len(args) > 64 {
		return Task{}, ErrBadVerifyArgs
	}
	for _, word := range args {
		if word == "" || len(word) > 1024 || strings.ContainsAny(word, "\x00\n\r") {
			return Task{}, ErrBadVerifyArgs
		}
	}
	action := args[1]
	allowed := false
	switch args[0] {
	case "go":
		allowed = action == "test" || action == "vet"
	case "npm", "pnpm", "yarn", "bun":
		// Named package scripts are allowed only for verification. A package
		// script can run arbitrary child processes, hence the OS read-only
		// filesystem + secretless/no-egress sandbox is mandatory.
		if action == "run" && len(args) >= 3 {
			action = args[2]
		}
		allowed = verifyScript(action)
	case "cargo":
		allowed = action == "test" || action == "check" || action == "clippy"
	case "make", "just":
		allowed = verifyScript(action)
	case "dotnet":
		allowed = action == "test"
	case "mvn", "mvnw", "gradle":
		allowed = action == "test" || action == "check" || action == "verify"
	case "npx":
		allowed = len(args) >= 3 && (action == "--no-install" || action == "--yes") &&
			(args[2] == "playwright" || args[2] == "vitest" ||
				args[2] == "eslint" || args[2] == "tsc" || args[2] == "svelte-check")
	case "uv", "poetry":
		allowed = action == "run" && len(args) >= 3 &&
			(args[2] == "pytest" || args[2] == "ruff" || args[2] == "mypy")
	case "pytest", "ruff", "eslint", "tsc", "biome", "gofmt", "golangci-lint", "svelte-check", "vitest", "jest", "playwright":
		allowed = true
	case "python", "python3":
		allowed = action == "-m" && len(args) >= 3 &&
			(args[2] == "pytest" || args[2] == "unittest" || args[2] == "ruff")
	}
	if !allowed {
		return Task{}, ErrBadVerifyArgs
	}
	for _, arg := range args[1:] {
		// Prevent well-known arbitrary-code and file-output switches; tests
		// still run code, but have no writable source or secrets.
		for _, bad := range []string{
			"--exec", "-exec", "-exec=", "--output=", "--out=", "--fix",
			"--write", "--update", "-w", "--updateSnapshot",
			"--config=", "--plugin=", "--require=",
		} {
			if arg == bad || strings.HasPrefix(arg, bad) {
				return Task{}, ErrBadVerifyArgs
			}
		}
	}
	return Task{
		Name: "verify", Description: "QA verification", Kind: "check",
		Command: append([]string(nil), args...), TimeoutSeconds: 240,
	}, nil
}

func verifyScript(action string) bool {
	switch action {
	case "test", "test:unit", "test:e2e", "test:integration", "lint",
		"check", "typecheck", "type-check", "verify", "format:check", "build:check":
		return true
	}
	return false
}
