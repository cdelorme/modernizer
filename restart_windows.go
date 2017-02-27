package modernizer

import (
	"os"
	"syscall"
	"unsafe"
)

var restart = func(binary string) error {
	if _, _, err := syscall.StartProcess(binary, append([]string{binary}, os.Args[1:]...), &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}); err != nil {
		return err
	}
	syscall.NewLazyDLL("kernel32.dll").NewProc("SetFileAttributesW").Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(binary))), 2)
	os.Exit(0)
	return nil
}
