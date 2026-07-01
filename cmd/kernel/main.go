//go:build js && wasm

package main

import (
	"github.com/hack-pad/hackpad/kernel"
)

func main() {
	kernel.Init()
	select {}
}
