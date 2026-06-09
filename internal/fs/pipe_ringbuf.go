//go:build !pipe_chan

package fs

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/hack-pad/hackpad/internal/interop"
)

// pipeImpl is a synchronized ring-buffer pipe.
// Close() signals EOF after the buffer is drained.
// Buffer capacity must be a power of 2 (currently 32 KiB).
type pipeImpl struct {
	buf            []byte
	start, end     int
	full           bool
	done           chan struct{}
	reader, writer FID
	mu             sync.Mutex
	cond           *sync.Cond
}

func newPipeImpl(reader, writer FID) pipeCore {
	p := &pipeImpl{
		buf:    make([]byte, 32<<10), // 32KiB, must be power of 2
		done:   make(chan struct{}),
		reader: reader,
		writer: writer,
	}
	p.cond = sync.NewCond(&p.mu)
	return p
}

// bufLen returns the number of readable bytes in the ring buffer.
func (p *pipeImpl) bufLen() int {
	if !p.full {
		return (p.end - p.start) & (len(p.buf) - 1)
	}
	return len(p.buf)
}

// bufFree returns the number of writable bytes in the ring buffer.
func (p *pipeImpl) bufFree() int {
	return len(p.buf) - p.bufLen()
}

// bufRead copies up to len(dst) bytes from the ring buffer into dst.
func (p *pipeImpl) bufRead(dst []byte) int {
	n := p.bufLen()
	if n > len(dst) {
		n = len(dst)
	}
	if n == 0 {
		return 0
	}

	if p.start+n <= len(p.buf) {
		copy(dst, p.buf[p.start:p.start+n])
	} else {
		first := len(p.buf) - p.start
		copy(dst, p.buf[p.start:])
		copy(dst[first:], p.buf[:n-first])
	}
	p.start = (p.start + n) & (len(p.buf) - 1)
	p.full = false
	return n
}

// bufWrite copies up to len(src) bytes from src into the ring buffer.
func (p *pipeImpl) bufWrite(src []byte) int {
	free := p.bufFree()
	if free > len(src) {
		free = len(src)
	}
	if free == 0 {
		return 0
	}

	if p.end+free <= len(p.buf) {
		copy(p.buf[p.end:p.end+free], src[:free])
	} else {
		first := len(p.buf) - p.end
		copy(p.buf[p.end:], src[:first])
		copy(p.buf[:free-first], src[first:free])
	}
	p.end = (p.end + free) & (len(p.buf) - 1)
	if p.end == p.start {
		p.full = true
	}
	return free
}

func (p *pipeImpl) Stat() (os.FileInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return &pipeStat{
		name: "",
		size: int64(p.bufLen()),
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

func (p *pipeImpl) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		if n := p.bufLen(); n > 0 {
			n = p.bufRead(b)
			p.cond.Broadcast()
			return n, nil
		}

		select {
		case <-p.done:
			if n := p.bufLen(); n > 0 {
				n = p.bufRead(b)
				p.cond.Broadcast()
				return n, nil
			}
			return 0, io.EOF
		default:
		}

		p.cond.Wait()
	}
}

func (p *pipeImpl) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	total := 0
	for total < len(b) {
		select {
		case <-p.done:
			if total > 0 {
				return total, nil
			}
			return 0, interop.BadFileNumber(p.writer)
		default:
		}

		n := p.bufWrite(b[total:])
		total += n
		if n > 0 {
			p.cond.Broadcast()
		}

		if total < len(b) {
			p.cond.Wait()
		}
	}
	return total, nil
}

func (p *pipeImpl) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.done:
		return interop.BadFileNumber(p.writer)
	default:
		close(p.done)
		p.cond.Broadcast()
		return nil
	}
}
