// +build !windows

package modernizer

import (
	"os"
	"syscall"
)

var restart = func(binary string) error {
	return syscall.Exec(binary, append([]string{binary}, os.Args[1:]...), os.Environ())
}
