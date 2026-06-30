//go:build js

package fs

import (
	"syscall/js"

	"fmt"

	"github.com/hack-pad/hackpad/internal/interop"
)

func link(args []js.Value) ([]interface{}, error) {
	_, err := linkSync(args)
	return nil, err
}

func linkSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Invalid number of args, expected 2: %v", args)
	}
	return nil, interop.ErrNotImplemented
}
