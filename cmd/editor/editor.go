//go:build js

package main

import (
	"os"
	"path/filepath"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/cmd/editor/ide"
	"github.com/hack-pad/hackpad/internal/log"
)

// editorJSFunc is a JS function that opens on a JS element and returns a JS object with the following spec:
//
//	{
//	  getContents() string
//	  setContents(string)
//	  getCursorIndex() int
//	  setCursorIndex(int)
//	}
type editorJSFunc js.Value

func (e editorJSFunc) New(elem *dom.Element) ide.Editor {
	editor := &jsEditor{
		titleChan: make(chan string, 1),
	}
	editor.elem = js.Value(e).Invoke(elem.JSValue(), js.FuncOf(editor.onEdit))
	return editor
}

type jsEditor struct {
	elem      js.Value
	filePath  string
	titleChan chan string
}

func (j *jsEditor) onEdit(js.Value, []js.Value) interface{} {
	go func() {
		contents := j.elem.Call("getContents").String()
		perm := os.FileMode(0700)
		info, err := os.Stat(j.filePath)
		if err == nil {
			perm = info.Mode()
		}
		err = os.WriteFile(j.filePath, []byte(contents), perm)
		if err != nil {
			log.Error("Failed to write file contents: ", err)
		}
	}()
	return nil
}

func (j *jsEditor) OpenFile(path string) error {
	j.filePath = path
	j.titleChan <- path
	j.setLanguage(path)
	return j.ReloadFile()
}

func (j *jsEditor) CurrentFile() string {
	return j.filePath
}

func (j *jsEditor) ReloadFile() error {
	contents, err := os.ReadFile(j.filePath)
	if err != nil {
		return err
	}
	j.elem.Call("setContents", string(contents))
	return nil
}

func (j *jsEditor) GetCursor() int {
	return j.elem.Call("getCursorIndex").Int()
}

func (j *jsEditor) SetCursor(i int) error {
	j.elem.Call("setCursorIndex", i)
	return nil
}

func (j *jsEditor) Titles() <-chan string {
	return j.titleChan
}

func languageFromExt(path string) string {
	switch filepath.Ext(path) {
	case ".go":
		return "go"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".json":
		return "json"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".md", ".markdown":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".sh", ".bash":
		return "shell"
	default:
		return "plaintext"
	}
}

func (j *jsEditor) setLanguage(path string) {
	j.elem.Call("setLanguage", languageFromExt(path))
}
