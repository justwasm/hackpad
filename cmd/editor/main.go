//go:build js

package main

import (
	"embed"
	"flag"
	"os"
	"path/filepath"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/cmd/editor/ide"
	"github.com/hack-pad/hackpad/cmd/editor/plaineditor"
	"github.com/hack-pad/hackpad/cmd/editor/taskconsole"
	"github.com/hack-pad/hackpad/cmd/editor/terminal"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
)

//go:embed example/*.txt
var exampleFS embed.FS

const (
	goBinaryPath = "go"
)

func writeExampleFile(fs embed.FS, src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil // already exists
	}
	data, err := fs.ReadFile(src)
	if err != nil {
		log.Error("Failed to read embedded "+filepath.Base(src)+": ", err)
		return err
	}
	if err := os.WriteFile(dst, data, 0600); err != nil {
		log.Error("Failed to write "+dst+": ", err)
		return err
	}
	return nil
}

func main() {
	editorID := flag.String("editor", "", "Editor element ID to attach")
	flag.Parse()

	if *editorID == "" {
		flag.Usage()
		os.Exit(2)
	}

	app := dom.GetDocument().GetElementByID(*editorID)
	app.AddClass("ide")
	globalEditorProps := js.Global().Get("editor")
	globalEditorProps.Set("profile", js.FuncOf(interop.ProfileJS))
	newEditor := globalEditorProps.Get("newEditor")
	var editorBuilder ide.EditorBuilder = editorJSFunc(newEditor)
	if !newEditor.Truthy() {
		editorBuilder = plaineditor.New()
	}
	newXTermFunc := globalEditorProps.Get("newTerminal")
	if !newXTermFunc.Truthy() {
		panic("window.editor.newTerminal must be set")
	}

	consoleBuilder := terminal.New(newXTermFunc)
	taskConsoleBuilder := taskconsole.New()
	win, tasks := ide.New(app, editorBuilder, consoleBuilder, taskConsoleBuilder)

	if _, err := tasks.Start(goBinaryPath, "go", "version"); err != nil {
		log.Error("Failed to start go version: ", err)
		return
	}

	if err := os.MkdirAll("playground", 0700); err != nil {
		log.Error("Failed to make playground dir", err)
		return
	}
	if err := os.Chdir("playground"); err != nil {
		log.Error("Failed to switch to playground dir", err)
		return
	}

	if err := writeExampleFile(exampleFS, "example/main.go.txt", "main.go"); err != nil {
		return
	}

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		_, err := tasks.Start(goBinaryPath, "go", "mod", "init", "playground")
		if err != nil {
			log.Error("Failed to start module init: ", err)
			return
		}
		_, err = tasks.Start(goBinaryPath, "go", "mod", "tidy")
		if err != nil {
			log.Error("Failed to start go mod tidy: ", err)
			return
		}
	}

	win.NewConsole()
	editor := win.NewEditor()
	if err := editor.OpenFile("main.go"); err != nil {
		log.Error("Failed to open main.go in editor: ", err)
	}

	select {}
}
