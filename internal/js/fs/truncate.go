//go:build js

package fs

import (
	"os"
	"syscall/js"

	"fmt"

	"github.com/hack-pad/hackpad/internal/process"
)

func truncate(args []js.Value) ([]interface{}, error) {
	_, err := truncateSync(args)
	return nil, err
}

func truncateSync(args []js.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("Invalid number of args, expected at least 1: %v", args)
	}
	path := args[0].String()
	length := int64(0)
	if len(args) >= 2 {
		length = int64(args[1].Int())
	}

	p := process.Current()
	// open with write-only (don't create if missing)
	fd, err := p.Files().Open(path, os.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}
	defer p.Files().Close(fd)
	return nil, p.Files().Truncate(fd, length)
}
