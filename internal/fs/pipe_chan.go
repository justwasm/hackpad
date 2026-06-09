//go:build pipe_chan

package fs

import (
	"io"
	"os"
	"time"

	"github.com/hack-pad/hackpad/internal/interop"
)

// pipeImpl is a channel-based pipe with reference counting.
// Close() signals EOF after the buffer is drained.
type pipeImpl struct {
	buf            chan byte
	done           chan struct{}
	reader, writer FID
}

func newPipeImpl(reader, writer FID) pipeCore {
	const maxPipeBuffer = 32 << 10 // 32KiB
	return &pipeImpl{
		buf:    make(chan byte, maxPipeBuffer),
		done:   make(chan struct{}),
		reader: reader,
		writer: writer,
	}
}

func (p *pipeImpl) Stat() (os.FileInfo, error) {
	return &pipeStat{
		name: "",
		size: int64(len(p.buf)),
		mode: os.ModeNamedPipe,
	}, nil
}

func (p *pipeImpl) Sync() error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-p.done:
		return nil
	case <-timer.C:
		return io.ErrNoProgress
	}
}

func (p *pipeImpl) Read(buf []byte) (n int, err error) {
	for n < len(buf) {
		b, ok := <-p.buf
		if !ok {
			err = io.EOF
			return
		}
		buf[n] = b
		n++
	}
	if n == 0 {
		err = io.EOF
	}
	return
}

func (p *pipeImpl) Write(buf []byte) (n int, err error) {
	for _, b := range buf {
		select {
		case <-p.done:
			return 0, interop.BadFileNumber(p.writer)
		case p.buf <- b:
			n++
		}
	}
	if n < len(buf) {
		err = io.ErrShortWrite
	}
	return
}

func (p *pipeImpl) Close() error {
	select {
	case <-p.done:
		return interop.BadFileNumber(p.writer)
	default:
		close(p.done)
		close(p.buf)
		return nil
	}
}
