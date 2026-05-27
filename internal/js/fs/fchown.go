//go:build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func fchown(args []js.Value) ([]interface{}, error) {
	_, err := fchownSync(args)
	return nil, err
}

func fchownSync(args []js.Value) (interface{}, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("Invalid number of args, expected 3: %v", args)
	}
	fid := common.FID(args[0].Int())
	uid := args[1].Int()
	gid := args[2].Int()
	p := process.Current()
	return nil, p.Files().Fchown(fid, uid, gid)
}
