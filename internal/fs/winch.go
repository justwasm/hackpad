package fs

import (
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpadfs"
)

// WinchSize holds terminal dimensions.
type WinchSize struct {
	Cols, Rows, Xpx, Ypx int
}

var releaseWinchCB = func(*winchReader) {}

// winchOpener is set by the JS-backed implementation (winch_js.go) when
// built with the js build tag. Its default returns ErrNotImplemented.
var winchOpener = defaultWinchOpener

func defaultWinchOpener(termID string, flags int) (hackpadfs.File, error) {
	return nil, interop.ErrNotImplemented
}

// winchManager tracks termID-to-broadcaster for the kernel side (editor
// terminal builder, setWinch routing). The actual cross-instance data
// sharing happens through JS globals and CustomEvents (winch_js.go).
type winchManager struct {
	mu        sync.Mutex
	terminals map[string]struct{}
	nextID    int
}

// WinchManager is the global terminal winch registry (kernel side).
var WinchManager = &winchManager{
	terminals: make(map[string]struct{}),
}

// Register marks a termID as valid (kernel only; child processes use JS).
func (m *winchManager) Register(termID string) {
	m.mu.Lock()
	m.terminals[termID] = struct{}{}
	m.mu.Unlock()
}

// Exists checks if a termID has been registered.
func (m *winchManager) Exists(termID string) bool {
	m.mu.Lock()
	_, ok := m.terminals[termID]
	m.mu.Unlock()
	return ok
}

// Remove removes a termID from the registry.
func (m *winchManager) Remove(termID string) {
	m.mu.Lock()
	delete(m.terminals, termID)
	m.mu.Unlock()
}

// NextTermID generates a unique terminal ID.
func (m *winchManager) NextTermID() string {
	m.mu.Lock()
	id := m.nextID
	m.nextID++
	m.mu.Unlock()
	return "term-" + strconv.Itoa(id)
}

// OpenWinch returns a virtual winch file for the given termID.
// Uses winchOpener which is backed by JS interop when built with js tag.
func (m *winchManager) OpenWinch(termID string, flags int) (hackpadfs.File, error) {
	return winchOpener(termID, flags)
}

// winchReader implements blocking read from a JS-backed winch data source.
// It reads initial data from window.__winch_{termID} and subscribes to
// CustomEvent("winch:{termID}", ...) for subsequent resize events.
type winchReader struct {
	termID  string
	mu      sync.Mutex
	ch      chan string // receives formatted "COLS ROWS XPX YPX\n" strings
	buf     []byte
	closed  bool
	cb      interface{} // js.Func; released via releaseWinchCB
	setup   func()
	setupDo sync.Once
}

func newWinchReader(termID string) *winchReader {
	return &winchReader{
		termID: termID,
		ch:     make(chan string, 8),
	}
}

func (r *winchReader) Read(p []byte) (int, error) {
	r.setupDo.Do(func() {
		r.setup()
	})
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return 0, io.EOF
	}
	if len(r.buf) > 0 {
		n := copy(p, r.buf)
		r.buf = r.buf[n:]
		r.mu.Unlock()
		return n, nil
	}
	r.mu.Unlock()

	s, ok := <-r.ch
	if !ok {
		return 0, io.EOF
	}
	n := copy(p, s)
	if n < len(s) {
		r.mu.Lock()
		r.buf = append([]byte(nil), s[n:]...)
		r.mu.Unlock()
	}
	return n, nil
}

func (r *winchReader) Write(p []byte) (int, error)           { return 0, interop.ErrNotImplemented }
func (r *winchReader) ReadAt(p []byte, off int64) (int, error) { return r.Read(p) }
func (r *winchReader) Seek(offset int64, whence int) (int64, error)  { return 0, nil }
func (r *winchReader) WriteAt(p []byte, off int64) (int, error)     { return 0, interop.ErrNotImplemented }
func (r *winchReader) Truncate(size int64) error                      { return interop.ErrNotImplemented }

func (r *winchReader) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()
	releaseWinchCB(r)
	close(r.ch)
	return nil
}

func (r *winchReader) Stat() (os.FileInfo, error) {
	return winchStat{name: "winch"}, nil
}

// winchWriter dispatches terminal dimensions via JS.
type winchWriter struct {
	termID string
}

func newWinchWriter(termID string) *winchWriter {
	return &winchWriter{termID: termID}
}

func (w *winchWriter) Write(p []byte) (int, error) {
	// Not used by CPU processes; writes go through setWinch/CustomEvent.
	return len(p), nil
}

func (w *winchWriter) Read(p []byte) (int, error)       { return 0, interop.ErrNotImplemented }
func (w *winchWriter) ReadAt(p []byte, off int64) (int, error)  { return 0, interop.ErrNotImplemented }
func (w *winchWriter) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (w *winchWriter) WriteAt(p []byte, off int64) (int, error)   { return w.Write(p) }
func (w *winchWriter) Truncate(size int64) error                    { return interop.ErrNotImplemented }
func (w *winchWriter) Close() error                                 { return nil }
func (w *winchWriter) Stat() (os.FileInfo, error) {
	return winchStat{name: "winch"}, nil
}

type winchStat struct {
	name string
}

func (s winchStat) Name() string       { return s.name }
func (s winchStat) Size() int64        { return 0 }
func (s winchStat) Mode() os.FileMode  { return os.ModeCharDevice | 0644 }
func (s winchStat) ModTime() time.Time { return time.Time{} }
func (s winchStat) IsDir() bool        { return false }
func (s winchStat) Sys() interface{}   { return nil }

// winchDataKey returns the JS global key for a termID.
func winchDataKey(termID string) string { return "__winch_" + termID }

// winchEventType returns the CustomEvent type for a termID.
func winchEventType(termID string) string { return "winch:" + termID }

// formatWinchStr serializes dimensions into "COLS ROWS XPX YPX\n".
func formatWinchStr(ws WinchSize) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(ws.Cols))
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(ws.Rows))
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(ws.Xpx))
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(ws.Ypx))
	b.WriteString("\n")
	return b.String()
}
