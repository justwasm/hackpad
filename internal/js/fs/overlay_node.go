//go:build js

package fs

import (
	"errors"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/nodefs"
	"github.com/hack-pad/hackpad/internal/promise"
)

func overlayNodefs(this js.Value, args []js.Value) interface{} {
	resolve, reject, prom := promise.New()
	go func() {
		err := OverlayNodefs(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed overlaying Node.js FS"))
		} else {
			log.Debug("Successfully overlayed Node.js FS")
			resolve(nil)
		}
	}()
	return prom.JSValue()
}

func OverlayNodefs(args []js.Value) error {
	if len(args) < 2 {
		return errors.New("overlayNodefs: mount path and local path are required")
	}
	mountPath := args[0].String()
	localPath := args[1].String()

	mode := "read"
	if len(args) >= 3 && args[2].Type() == js.TypeString {
		mode = args[2].String()
	}

	nfs, err := nodefs.NewFS(localPath, mode)
	if err != nil {
		return err
	}
	return fs.Overlay(mountPath, nfs)
}
