//go:build js

package nodefs

import (
	"io"
	"sync"
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpadfs"
)

// nodefsFile implements hackpadfs.File with an in-memory buffer,
// flushed to disk on Sync/Close.
type nodefsFile struct {
	fs       *FS
	name     string
	resolved string
	info     hackpadfs.FileInfo
	buf      []byte
	offset   int64
	ro       bool
	append   bool
	closed   bool
	dirty    bool

	mu sync.Mutex
}

var (
	_ hackpadfs.File           = (*nodefsFile)(nil)
	_ hackpadfs.ReadWriterFile = (*nodefsFile)(nil)
	_ hackpadfs.SeekerFile     = (*nodefsFile)(nil)
	_ hackpadfs.TruncaterFile  = (*nodefsFile)(nil)
	_ hackpadfs.SyncerFile     = (*nodefsFile)(nil)
)

func (f *nodefsFile) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, hackpadfs.ErrClosed
	}
	if f.offset >= int64(len(f.buf)) {
		return 0, io.EOF
	}
	n := copy(p, f.buf[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *nodefsFile) ReadAt(p []byte, off int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, hackpadfs.ErrClosed
	}
	if off >= int64(len(f.buf)) {
		return 0, io.EOF
	}
	n := copy(p, f.buf[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (f *nodefsFile) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, hackpadfs.ErrClosed
	}
	if f.ro {
		return 0, &hackpadfs.PathError{Op: "write", Path: f.name, Err: hackpadfs.ErrPermission}
	}
	if f.append {
		f.offset = int64(len(f.buf))
	}
	need := f.offset + int64(len(p))
	if need > int64(len(f.buf)) {
		newBuf := make([]byte, need)
		copy(newBuf, f.buf)
		f.buf = newBuf
	}
	n := copy(f.buf[f.offset:], p)
	f.offset += int64(n)
	f.dirty = true
	return n, nil
}

func (f *nodefsFile) Seek(offset int64, whence int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, hackpadfs.ErrClosed
	}
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(f.buf)) + offset
	default:
		return 0, &hackpadfs.PathError{Op: "seek", Path: f.name, Err: hackpadfs.ErrInvalid}
	}
	if newOffset < 0 {
		return 0, &hackpadfs.PathError{Op: "seek", Path: f.name, Err: hackpadfs.ErrInvalid}
	}
	f.offset = newOffset
	return f.offset, nil
}

func (f *nodefsFile) Truncate(size int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return hackpadfs.ErrClosed
	}
	if f.ro {
		return &hackpadfs.PathError{Op: "truncate", Path: f.name, Err: hackpadfs.ErrPermission}
	}
	if size < int64(len(f.buf)) {
		f.buf = f.buf[:size]
	} else if size > int64(len(f.buf)) {
		newBuf := make([]byte, size)
		copy(newBuf, f.buf)
		f.buf = newBuf
	}
	f.dirty = true
	if f.offset > size {
		f.offset = size
	}
	return nil
}

func (f *nodefsFile) Sync() error {
	return f.flush()
}

func (f *nodefsFile) Stat() (hackpadfs.FileInfo, error) {
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if closed {
		return nil, hackpadfs.ErrClosed
	}
	// Re-stat from actual fs for latest size
	info, err := f.fs.Stat(f.name)
	if err == nil {
		f.mu.Lock()
		f.info = info
		f.mu.Unlock()
		return info, nil
	}
	f.mu.Lock()
	info = f.info
	f.mu.Unlock()
	return info, nil
}

func (f *nodefsFile) Close() error {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return hackpadfs.ErrClosed
	}
	f.closed = true
	dirty := f.dirty
	f.mu.Unlock()

	if dirty {
		return f.flush()
	}
	return nil
}

func (f *nodefsFile) flush() error {
	f.mu.Lock()
	buf := f.buf
	resolved := f.resolved
	dirty := f.dirty
	f.dirty = false
	f.mu.Unlock()

	if !dirty {
		return nil
	}

	bridge := f.fs.bridge()
	jsBuf := js.Global().Get("Uint8Array").New(len(buf))
	if len(buf) > 0 {
		js.CopyBytesToJS(jsBuf, buf)
	}
	result := bridge.Call("writeFileSync", resolved, jsBuf)
	return bridgePathErr("write", f.name, result)
}

func (f *nodefsFile) ReadDir(n int) ([]hackpadfs.DirEntry, error) {
	return nil, &hackpadfs.PathError{Op: "readdir", Path: f.name, Err: hackpadfs.ErrIsDir}
}

// timeFromMs converts a JavaScript milliseconds timestamp to time.Time.
func timeFromMs(ms float64) time.Time {
	return time.Unix(0, int64(ms)*int64(time.Millisecond))
}
