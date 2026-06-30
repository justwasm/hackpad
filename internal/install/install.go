//go:build js

package install

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/hack-pad/hackpad/internal/promise"
)

func InstallFunc(this js.Value, args []js.Value) any {
	resolve, reject, prom := promise.New()
	go func() {
		err := Install(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed to install binary"))
			return
		}
		resolve(nil)
	}()
	return prom.JSValue()
}

func Install(args []js.Value) error {
	if len(args) == 0 || len(args) > 2 {
		return errors.New("Expected url to install, and optionally a binary name")
	}
	url := args[0].String()
	var command string
	if len(args) == 2 {
		command = args[1].String()
	} else {
		command = strings.TrimSuffix(filepath.Base(url), ".wasm")
	}

	if err := os.MkdirAll("/bin", 0644); err != nil {
		return err
	}

	body, err := httpGetFetch(url)
	if err != nil {
		return err
	}
	defer runtime.GC()
	fs := process.Current().Files()
	fd, err := fs.Open("/bin/"+command, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0750)
	if err != nil {
		return err
	}
	defer fs.Close(fd)
	if _, err := fs.Write(fd, body, 0, body.Len(), nil); err != nil {
		return err
	}
	log.Print("Install completed: ", command)
	return nil
}
