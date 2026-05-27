//go:build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

// symlink(path, link, callback) — 'path' is the target, 'link' is the symlink location
func symlink(args []js.Value) ([]interface{}, error) {
	_, err := symlinkSync(args)
	return nil, err
}

func symlinkSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	target := args[0].String()
	newname := args[1].String()
	p := process.Current()
	return nil, p.Files().Symlink(target, newname)
}
