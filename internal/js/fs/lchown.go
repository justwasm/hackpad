//go:build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func lchown(args []js.Value) ([]interface{}, error) {
	_, err := lchownSync(args)
	return nil, err
}

func lchownSync(args []js.Value) (interface{}, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("Invalid number of args, expected 3: %v", args)
	}
	// no-op: user/group ID not supported on wasm
	return nil, nil
}
