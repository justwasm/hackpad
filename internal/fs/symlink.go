package fs

import "github.com/hack-pad/hackpadfs"

// ReadlinkFS is implemented by file systems that support reading symlink targets.
type ReadlinkFS interface {
hackpadfs.FS
Readlink(name string) (string, error)
}
