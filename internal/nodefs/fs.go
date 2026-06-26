//go:build js

// Package nodefs implements hackpadfs.FS backed by Node.js's native 'fs' module,
// accessed via a pre-injected globalThis.__nodefs bridge object.
//
// Usage (Node.js host):
//
//	const fs = require('fs');
//	const path = require('path');
//
//	function statToObj(s) {
//	  return {
//	    dev: s.dev, ino: s.ino, mode: s.mode, nlink: s.nlink,
//	    uid: s.uid, gid: s.gid, rdev: s.rdev, size: s.size,
//	    blksize: s.blksize, blocks: s.blocks,
//	    isDirectory: s.isDirectory(),
//	    isFile: s.isFile(),
//	    isSymbolicLink: s.isSymbolicLink(),
//	    atimeMs: s.atimeMs, mtimeMs: s.mtimeMs, ctimeMs: s.ctimeMs,
//	  };
//	}
//
//	globalThis.__nodefs = {
//	  readFileSync(p) { const b = fs.readFileSync(p); return { data: new Uint8Array(b).buffer, err: null }; },
//	  writeFileSync(p, buf) { fs.writeFileSync(p, Buffer.from(buf)); return { err: null }; },
//	  statSync(p) { const s = fs.statSync(p); return { stat: statToObj(s), err: null }; },
//	  lstatSync(p) { const s = fs.lstatSync(p); return { stat: statToObj(s), err: null }; },
//	  mkdirSync(p, mode) { fs.mkdirSync(p, { mode }); return { err: null }; },
//	  mkdirAllSync(p, mode) { fs.mkdirSync(p, { recursive: true, mode }); return { err: null }; },
//	  readdirSync(p) { const entries = fs.readdirSync(p, { withFileTypes: true }); return { entries: entries.map(d => ({ name: d.name, isDirectory: d.isDirectory(), isSymbolicLink: d.isSymbolicLink() })), err: null }; },
//	  unlinkSync(p) { fs.unlinkSync(p); return { err: null }; },
//	  rmdirSync(p) { fs.rmdirSync(p); return { err: null }; },
//	  rmSync(p, recursive) { fs.rmSync(p, { recursive, force: true }); return { err: null }; },
//	  renameSync(oldP, newP) { fs.renameSync(oldP, newP); return { err: null }; },
//	  chmodSync(p, mode) { fs.chmodSync(p, mode); return { err: null }; },
//	  utimesSync(p, atime, mtime) { fs.utimesSync(p, atime, mtime); return { err: null }; },
//	  symlinkSync(target, p) { fs.symlinkSync(target, p); return { err: null }; },
//	  readlinkSync(p) { return { link: fs.readlinkSync(p), err: null }; },
//	  accessSync(p, mode) { fs.accessSync(p, mode); return { err: null }; },
//	};
package nodefs

import (
	"errors"
	"path"
	"sync"
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpadfs"
)

// FS implements hackpadfs.FS backed by Node.js's native fs module.
type FS struct {
	root string
	mode string // "read" or "readwrite"

	mu     sync.Mutex
	openWriters map[string]bool // tracks files with pending writes
}

// NewFS creates a new nodefs FS rooted at the given local path.
// mode must be "read" or "readwrite".
func NewFS(root string, mode string) (*FS, error) {
	if root == "" {
		return nil, errors.New("nodefs: root path is required")
	}
	// Verify the Node.js bridge exists
	bridge := js.Global().Get("__nodefs")
	if bridge.IsUndefined() || bridge.IsNull() {
		return nil, errors.New("nodefs: globalThis.__nodefs is not defined — inject the Node.js fs bridge before loading WASM")
	}
	return &FS{
		root:        root,
		mode:        mode,
		openWriters: make(map[string]bool),
	}, nil
}

func (f *FS) resolve(name string) string {
	if path.IsAbs(name) {
		return path.Join(f.root, name)
	}
	return path.Join(f.root, name)
}

func (f *FS) ensureWrite() error {
	if f.mode != "readwrite" {
		return hackpadfs.ErrPermission
	}
	return nil
}

// bridge returns the __nodefs JS object.
func (f *FS) bridge() js.Value {
	return js.Global().Get("__nodefs")
}

// --- hackpadfs.FS ---

// Open implements hackpadfs.FS
func (f *FS) Open(name string) (hackpadfs.File, error) {
	return f.OpenFile(name, hackpadfs.FlagReadOnly, 0)
}

// OpenFile implements hackpadfs.OpenFileFS
func (f *FS) OpenFile(name string, flag int, perm hackpadfs.FileMode) (hackpadfs.File, error) {
	resolved := f.resolve(name)

	// Determine if we need write access
	writeFlags := hackpadfs.FlagWriteOnly | hackpadfs.FlagReadWrite | hackpadfs.FlagCreate | hackpadfs.FlagAppend | hackpadfs.FlagTruncate
	if flag&writeFlags != 0 {
		if err := f.ensureWrite(); err != nil {
			return nil, &hackpadfs.PathError{Op: "open", Path: name, Err: err}
		}
	}

	// Check if file exists
	bridge := f.bridge()
	var fileBuf []byte
	isNew := false

	accessErr := callBridge(bridge, "accessSync", resolved, 0)
	if accessErr != nil {
		if flag&hackpadfs.FlagCreate != 0 {
			// Create new file: start with empty buffer
			isNew = true
		} else {
			return nil, &hackpadfs.PathError{Op: "open", Path: name, Err: hackpadfs.ErrNotExist}
		}
	}

	if !isNew {
		// Read entire file into memory
		result := bridge.Call("readFileSync", resolved)
		if err := bridgeErr(result); err != nil {
			return nil, &hackpadfs.PathError{Op: "open", Path: name, Err: err}
		}
		if data := result.Get("data"); !data.IsUndefined() && data.Type() == js.TypeObject {
			jsBuf := js.Global().Get("Uint8Array").New(data)
			fileBuf = make([]byte, jsBuf.Length())
			js.CopyBytesToGo(fileBuf, jsBuf)
		}
	}

	if flag&hackpadfs.FlagTruncate != 0 {
		fileBuf = nil
	}

	finfo, err := f.Stat(name)
	if err != nil && !isNew {
		return nil, &hackpadfs.PathError{Op: "open", Path: name, Err: err}
	}
	if isNew {
		finfo = &nodefsFileInfo{
			name:    name,
			size:    0,
			mode:    perm | hackpadfs.ModePerm,
			isDir:   false,
			modTime: time.Now(),
		}
	}

	nf := &nodefsFile{
		fs:       f,
		name:     name,
		resolved: resolved,
		info:     finfo,
		buf:      fileBuf,
		ro:       flag&writeFlags == 0,
		append:   flag&hackpadfs.FlagAppend != 0,
	}
	if nf.append && fileBuf != nil {
		nf.offset = int64(len(fileBuf))
	}
	return nf, nil
}

// Stat implements hackpadfs.StatFS
func (f *FS) Stat(name string) (hackpadfs.FileInfo, error) {
	return f.stat(name, false)
}

// Lstat implements hackpadfs.LstatFS
func (f *FS) Lstat(name string) (hackpadfs.FileInfo, error) {
	return f.stat(name, true)
}

func (f *FS) stat(name string, useLstat bool) (hackpadfs.FileInfo, error) {
	resolved := f.resolve(name)
	bridge := f.bridge()

	var result js.Value
	if useLstat {
		result = bridge.Call("lstatSync", resolved)
	} else {
		result = bridge.Call("statSync", resolved)
	}
	if err := bridgeErr(result); err != nil {
		return nil, &hackpadfs.PathError{Op: "stat", Path: name, Err: err}
	}

	statObj := result.Get("stat")
	return &nodefsFileInfo{
		name:    name,
		size:    int64(statObj.Get("size").Float()),
		mode:    hackpadfs.FileMode(statObj.Get("mode").Int()),
		isDir:   statObj.Get("isDirectory").Bool(),
		modTime: time.UnixMilli(int64(statObj.Get("mtimeMs").Float())),
	}, nil
}

// Mkdir implements hackpadfs.MkdirFS
func (f *FS) Mkdir(name string, perm hackpadfs.FileMode) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "mkdir", Path: name, Err: err}
	}
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("mkdirSync", resolved, int(perm))
	return bridgePathErr("mkdir", name, result)
}

// MkdirAll implements hackpadfs.MkdirAllFS
func (f *FS) MkdirAll(path_ string, perm hackpadfs.FileMode) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "mkdir", Path: path_, Err: err}
	}
	resolved := f.resolve(path_)
	bridge := f.bridge()
	result := bridge.Call("mkdirAllSync", resolved, int(perm))
	return bridgePathErr("mkdir", path_, result)
}

// Remove implements hackpadfs.RemoveFS
func (f *FS) Remove(name string) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "remove", Path: name, Err: err}
	}
	resolved := f.resolve(name)
	bridge := f.bridge()

	// Try unlink first, then rmdir
	result := bridge.Call("unlinkSync", resolved)
	if bridgeErr(result) == nil {
		return nil
	}
	result = bridge.Call("rmdirSync", resolved)
	return bridgePathErr("remove", name, result)
}

// RemoveAll implements hackpadfs.RemoveAllFS
func (f *FS) RemoveAll(name string) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "removeall", Path: name, Err: err}
	}
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("rmSync", resolved, true)
	return bridgePathErr("removeall", name, result)
}

// Rename implements hackpadfs.RenameFS
func (f *FS) Rename(oldName, newName string) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "rename", Path: oldName, Err: err}
	}
	oldResolved := f.resolve(oldName)
	newResolved := f.resolve(newName)
	bridge := f.bridge()
	result := bridge.Call("renameSync", oldResolved, newResolved)
	return bridgePathErr("rename", oldName, result)
}

// Chmod implements hackpadfs.ChmodFS
func (f *FS) Chmod(name string, mode hackpadfs.FileMode) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "chmod", Path: name, Err: err}
	}
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("chmodSync", resolved, int(mode))
	return bridgePathErr("chmod", name, result)
}

// Chtimes implements hackpadfs.ChtimesFS
func (f *FS) Chtimes(name string, atime, mtime time.Time) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "chtimes", Path: name, Err: err}
	}
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("utimesSync", resolved, atime.UnixMilli(), mtime.UnixMilli())
	return bridgePathErr("chtimes", name, result)
}

// ReadDir implements hackpadfs.ReadDirFS
func (f *FS) ReadDir(name string) ([]hackpadfs.DirEntry, error) {
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("readdirSync", resolved)
	if err := bridgeErr(result); err != nil {
		return nil, &hackpadfs.PathError{Op: "readdir", Path: name, Err: err}
	}
	entriesVal := result.Get("entries")
	if entriesVal.IsUndefined() || entriesVal.Type() != js.TypeObject {
		return nil, &hackpadfs.PathError{Op: "readdir", Path: name, Err: errors.New("invalid readdir response")}
	}
	length := entriesVal.Get("length").Int()
	entries := make([]hackpadfs.DirEntry, 0, length)
	for i := 0; i < length; i++ {
		entry := entriesVal.Index(i)
		entries = append(entries, &nodefsDirEntry{
			name:        entry.Get("name").String(),
			isDirectory: entry.Get("isDirectory").Bool(),
			isSymlink:   entry.Get("isSymbolicLink").Bool(),
		})
	}
	return entries, nil
}

// Symlink implements hackpadfs.SymlinkFS
func (f *FS) Symlink(target, name string) error {
	if err := f.ensureWrite(); err != nil {
		return &hackpadfs.PathError{Op: "symlink", Path: name, Err: err}
	}
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("symlinkSync", target, resolved)
	return bridgePathErr("symlink", name, result)
}

// Readlink implements hackpadfs.ReadlinkFS
func (f *FS) Readlink(name string) (string, error) {
	resolved := f.resolve(name)
	bridge := f.bridge()
	result := bridge.Call("readlinkSync", resolved)
	if err := bridgeErr(result); err != nil {
		return "", &hackpadfs.PathError{Op: "readlink", Path: name, Err: err}
	}
	return result.Get("link").String(), nil
}

// --- helpers ---

func callBridge(bridge js.Value, fn string, args ...interface{}) error {
	result := bridge.Call(fn, args...)
	return bridgeErr(result)
}

func bridgeErr(result js.Value) error {
	errVal := result.Get("err")
	if errVal.IsUndefined() || errVal.IsNull() || errVal.Type() == js.TypeNull {
		return nil
	}
	return errors.New(errVal.String())
}

func bridgePathErr(op, path string, result js.Value) error {
	if err := bridgeErr(result); err != nil {
		return &hackpadfs.PathError{Op: op, Path: path, Err: err}
	}
	return nil
}

// --- file info ---

type nodefsFileInfo struct {
	name    string
	size    int64
	mode    hackpadfs.FileMode
	isDir   bool
	modTime time.Time
}

func (fi *nodefsFileInfo) Name() string      { return fi.name }
func (fi *nodefsFileInfo) Size() int64        { return fi.size }
func (fi *nodefsFileInfo) Mode() hackpadfs.FileMode { return fi.mode }
func (fi *nodefsFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *nodefsFileInfo) IsDir() bool        { return fi.isDir }
func (fi *nodefsFileInfo) Sys() interface{}   { return nil }

// --- dir entry ---

type nodefsDirEntry struct {
	name        string
	isDirectory bool
	isSymlink   bool
}

func (d *nodefsDirEntry) Name() string               { return d.name }
func (d *nodefsDirEntry) IsDir() bool                 { return d.isDirectory }
func (d *nodefsDirEntry) Type() hackpadfs.FileMode {
	if d.isDirectory {
		return hackpadfs.ModeDir
	}
	return 0
}
func (d *nodefsDirEntry) Info() (hackpadfs.FileInfo, error) {
	return nil, errors.New("nodefs: Info() not implemented without an fs reference")
}

// Ensure interfaces are satisfied
var (
	_ hackpadfs.FS           = (*FS)(nil)
	_ hackpadfs.OpenFileFS   = (*FS)(nil)
	_ hackpadfs.StatFS       = (*FS)(nil)
	_ hackpadfs.LstatFS      = (*FS)(nil)
	_ hackpadfs.MkdirFS      = (*FS)(nil)
	_ hackpadfs.MkdirAllFS   = (*FS)(nil)
	_ hackpadfs.RemoveFS     = (*FS)(nil)
	_ hackpadfs.RemoveAllFS  = (*FS)(nil)
	_ hackpadfs.RenameFS     = (*FS)(nil)
	_ hackpadfs.ChmodFS      = (*FS)(nil)
	_ hackpadfs.ChtimesFS    = (*FS)(nil)
	_ hackpadfs.ReadDirFS    = (*FS)(nil)
	_ hackpadfs.SymlinkFS    = (*FS)(nil)
	_ hackpadfs.ReadlinkFS   = (*FS)(nil)
	_ hackpadfs.FileInfo     = (*nodefsFileInfo)(nil)
	_ hackpadfs.DirEntry     = (*nodefsDirEntry)(nil)
)
