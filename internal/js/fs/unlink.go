//go:build js

package fs

import (
	"syscall/js"

	"fmt"

	"github.com/hack-pad/hackpad/internal/process"
)

func unlink(args []js.Value) ([]interface{}, error) {
	_, err := unlinkSync(args)
	return nil, err
}

func unlinkSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	p := process.Current()
	return nil, p.Files().Unlink(path)
}
