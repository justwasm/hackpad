//go:build js

package fs

import (
	"syscall/js"

	"fmt"

	"github.com/hack-pad/hackpad/internal/process"
)

// readlink(path, callback) — reads the target of a symlink
func readlink(args []js.Value) ([]interface{}, error) {
	target, err := readlinkSync(args)
	return []interface{}{target}, err
}

func readlinkSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	p := process.Current()
	return p.Files().Readlink(path)
}
