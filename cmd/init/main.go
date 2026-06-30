//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/install"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/js/fs"
	"github.com/hack-pad/hackpad/internal/js/process"
)

func init() {
	process.Init()
	fs.Init()
	global.Set("install", js.FuncOf(install.InstallFunc))
	interop.SetInitialized()
}

func main() {
	select {}
}
