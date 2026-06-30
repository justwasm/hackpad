//go:build js

package ide

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/css"
	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/promise"
)

var (
	//go:embed settings.html
	settingsHTML string
	//go:embed settings.css
	settingsCSS string
)

func init() {
	css.Add(settingsCSS)
}

func newSettings() *dom.Element {
	button := dom.New("button")
	button.SetInnerHTML(`<span class="fa fa-cog"></span>`)
	button.SetAttribute("className", "control")
	button.SetAttribute("title", "Settings")

	drop := newSettingsDropdown(button)
	button.AddEventListener("click", func(event js.Value) {
		event.Call("stopPropagation")
		drop.Toggle()
	})
	return button
}

func newSettingsDropdown(attachTo *dom.Element) *dropdown {
	elem := dom.New("div")
	elem.SetInnerHTML(settingsHTML)
	elem.AddClass("settings-dropdown")
	drop := newDropdown(attachTo, elem)

	listenButton := func(name, prompt string, fn func()) {
		elem.
			QuerySelector(fmt.Sprintf("button[title=%q]", name)).
			AddEventListener("click", func(event js.Value) {
				if prompt == "" || dom.Confirm(prompt) {
					go fn()
				}
			})
	}

	destroyMount := func(path string) promise.Promise {
		return promise.From(global.Get("destroyMount").Invoke(path))
	}
	listenButton("reset", "Erase all data and reload?", func() {
		mounts := interop.StringsFromJSValue(global.Get("getMounts").Invoke())
		var promises []promise.Promise
		for _, mount := range mounts {
			promises = append(promises, destroyMount(mount))
		}
		for _, p := range promises {
			_, _ = p.Await()
		}
		dom.Reload()
	})
	listenButton("clean build cache", "", func() {
		cache, err := os.UserCacheDir()
		if err == nil {
			destroyMount(cache)
		}
	})
	listenButton("mount local dir", "", func() {
		// Show directory picker with read-write access
		pickerPromise := js.Global().Call("showDirectoryPicker", map[string]any{"mode": "readwrite"})
		result, err := promise.From(pickerPromise).Await()
		if err != nil {
			log.Errorf("Failed to pick directory: %v", err)
			return
		}
		handle := result.(js.Value)
		dirName := handle.Get("name").String()
		mountPath := "/home/me/" + dirName

		// Check if this exact path is already mounted
		existingMounts := interop.StringsFromJSValue(global.Get("getMounts").Invoke())
		for _, m := range existingMounts {
			if m == mountPath || m == strings.TrimPrefix(mountPath, "/") {
				dom.Alert("Directory \"" + dirName + "\" is already mounted at " + mountPath + ".")
				return
			}
		}

		// Create mount directory
		if err := os.MkdirAll(mountPath, 0755); err != nil {
			log.Errorf("Failed to create mount directory %q: %v", mountPath, err)
			return
		}

		// Mount the directory with read-write access
		overlayFn := js.Global().Get("hackpad").Get("overlayLocalDir")
		if overlayFn.IsUndefined() {
			log.Errorf("overlayLocalDir not found on window.hackpad")
			return
		}
		_, err = promise.From(overlayFn.Invoke(mountPath, handle, "readwrite")).Await()
		if err != nil {
			log.Errorf("Failed to mount directory %q at %q: %v", dirName, mountPath, err)
			return
		}
		log.Debugf("Mounted local dir %q at %q", dirName, mountPath)
	})
	listenButton("reload programs", "Reinstall programs and reload?", func() {
		_, _ = destroyMount("/bin").Await()
		dom.Reload()
	})
	return drop
}
