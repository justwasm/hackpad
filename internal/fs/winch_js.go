//go:build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpadfs"
)

func init() {
	winchOpener = jsWinchOpener
	releaseWinchCB = func(r *winchReader) {
		if r.cb != nil {
			r.cb.(js.Func).Release()
		}
	}
}

// DispatchWinch writes to a JS global and dispatches a per-terminal
// CustomEvent so that JS-backed winch readers (in any wasm instance)
// receive the resize event.
func DispatchWinch(termID string, cols, rows, xpx, ypx int) {
	s := formatWinchStr(WinchSize{Cols: cols, Rows: rows, Xpx: xpx, Ypx: ypx})
	js.Global().Set(winchDataKey(termID), s)
	detail := js.Global().Get("Array").New()
	detail.Call("push", cols, rows, xpx, ypx)
	js.Global().Get("window").Call("dispatchEvent",
		js.Global().Get("CustomEvent").New(winchEventType(termID),
			js.ValueOf(map[string]interface{}{"detail": detail})))
}

func jsWinchOpener(termID string, flags int) (hackpadfs.File, error) {
	switch flags & 0x3 {
	case os.O_WRONLY, os.O_RDWR:
		return newWinchWriter(termID), nil
	default:
		r := newWinchReader(termID)
		r.setup = r.setupJS
		// Seed with initial value from JS global (set by kernel on terminal creation).
		r.feedFromJSGlobal()
		return r, nil
	}
}

// setupJS registers a per-terminal CustomEvent listener on window.
// Called once on first Read().
func (r *winchReader) setupJS() {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		detail := args[0].Get("detail")
		if detail.IsUndefined() || detail.IsNull() || detail.Length() < 2 {
			return nil
		}
		cols := detail.Index(0).Int()
		rows := detail.Index(1).Int()
		xpx := 0
		ypx := 0
		if detail.Length() >= 3 {
			xpx = detail.Index(2).Int()
		}
		if detail.Length() >= 4 {
			ypx = detail.Index(3).Int()
		}
		s := formatWinchStr(WinchSize{Cols: cols, Rows: rows, Xpx: xpx, Ypx: ypx})
		js.Global().Set(winchDataKey(r.termID), s)
		select {
		case r.ch <- s:
		default:
		}
		return nil
	})
	r.cb = cb
	js.Global().Get("window").Call("addEventListener", winchEventType(r.termID), cb)
}

// feedFromJSGlobal reads the latest cached value from window.__winch_{termID}.
// This is set by the kernel's setWinch on every resize.
func (r *winchReader) feedFromJSGlobal() {
	v := js.Global().Get(winchDataKey(r.termID))
	if v.IsUndefined() || v.IsNull() {
		return
	}
	s := v.String()
	if s != "" {
		select {
		case r.ch <- s:
		default:
		}
	}
}
