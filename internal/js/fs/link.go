//go:build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/pkg/errors"
)

func link(args []js.Value) ([]interface{}, error) {
	_, err := linkSync(args)
	return nil, err
}

func linkSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	return nil, interop.ErrNotImplemented
}
