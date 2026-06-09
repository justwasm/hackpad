package fs

import (
	"os"
	"time"

	"github.com/hack-pad/hackpad/internal/interop"
)

func (f *FileDescriptors) Pipe() [2]FID {
	r, w := newPipe(f.newFID)
	f.addFileDescriptor(r)
	f.addFileDescriptor(w)
	r.Open(f.parentPID)
	w.Open(f.parentPID)
	return [2]FID{r.id, w.id}
}

type pipeCore interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	Stat() (os.FileInfo, error)
	Sync() error
}

func newPipe(newFID func() FID) (r, w *fileDescriptor) {
	readerFID, writerFID := newFID(), newFID()
	pipeC := newPipeImpl(readerFID, writerFID)
	rPipe := &namedPipe{pipe: pipeC, fid: readerFID}
	r = newIrregularFileDescriptor(
		readerFID,
		rPipe.Name(),
		&pipeReadOnly{rPipe},
		os.ModeNamedPipe,
	)
	wPipe := &namedPipe{pipe: pipeC, fid: writerFID}
	w = newIrregularFileDescriptor(
		writerFID,
		wPipe.Name(),
		&pipeWriteOnly{wPipe},
		os.ModeNamedPipe,
	)
	return
}

type pipeStat struct {
	name string
	size int64
	mode os.FileMode
}

func (p pipeStat) Name() string       { return p.name }
func (p pipeStat) Size() int64        { return p.size }
func (p pipeStat) Mode() os.FileMode  { return p.mode }
func (p pipeStat) ModTime() time.Time { return time.Time{} }
func (p pipeStat) IsDir() bool        { return false }
func (p pipeStat) Sys() interface{}   { return nil }

type namedPipe struct {
	pipe pipeCore
	fid  FID
}

func (n *namedPipe) Read(buf []byte) (int, error)  { return n.pipe.Read(buf) }
func (n *namedPipe) Write(buf []byte) (int, error) { return n.pipe.Write(buf) }
func (n *namedPipe) Close() error                  { return n.pipe.Close() }
func (n *namedPipe) Stat() (os.FileInfo, error)    { return n.pipe.Stat() }
func (n *namedPipe) Sync() error                   { return n.pipe.Sync() }

func (n *namedPipe) Name() string {
	return "pipe" + n.fid.String()
}

type pipeReadOnly struct {
	*namedPipe
}

func (r *pipeReadOnly) ReadAt(buf []byte, off int64) (n int, err error) {
	if off == 0 {
		return r.Read(buf)
	}
	return 0, interop.ErrNotImplemented
}

func (r *pipeReadOnly) Write(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (r *pipeReadOnly) Close() error {
	// only write side of pipe should close the buffer
	return nil
}

type pipeWriteOnly struct {
	*namedPipe
}

func (w *pipeWriteOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (w *pipeWriteOnly) WriteAt(buf []byte, off int64) (n int, err error) {
	if off == 0 {
		return w.Write(buf)
	}
	return 0, interop.ErrNotImplemented
}
