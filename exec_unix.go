//go:build !windows

package main

import (
	"os"
	"syscall"
)

func syscallExec(path string) {
	syscall.Exec(path, []string{"claude"}, os.Environ())
}
