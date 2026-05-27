package fs

import (
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hack-pad/hackpadfs"
)

const maxSymlinkDepth = 40

// symlinkMap is a global store shared across all FileDescriptors instances.
// Symlinks are filesystem-level entities visible to all processes, so the
// store must be global (not per-process) to allow cross-process visibility.
var (
	symlinksMu sync.RWMutex
	symlinkMap = make(map[string]string)
)

func storeSymlink(symlinkPath, target string) {
	symlinksMu.Lock()
	symlinkMap[symlinkPath] = target
	symlinksMu.Unlock()
}

func lookupSymlink(symlinkPath string) (string, bool) {
	symlinksMu.RLock()
	target, ok := symlinkMap[symlinkPath]
	symlinksMu.RUnlock()
	return target, ok
}

func deleteSymlink(symlinkPath string) {
	symlinksMu.Lock()
	delete(symlinkMap, symlinkPath)
	symlinksMu.Unlock()
}

// symlinkFileInfo implements os.FileInfo for a symlink entry.
type symlinkFileInfo struct {
	name   string
	target string
}

func newSymlinkFileInfo(p, target string) os.FileInfo {
	return &symlinkFileInfo{name: path.Base(p), target: target}
}

func (s *symlinkFileInfo) Name() string      { return s.name }
func (s *symlinkFileInfo) Size() int64       { return int64(len(s.target)) }
func (s *symlinkFileInfo) Mode() os.FileMode { return os.ModeSymlink | 0777 }
func (s *symlinkFileInfo) ModTime() time.Time { return time.Time{} }
func (s *symlinkFileInfo) IsDir() bool       { return false }
func (s *symlinkFileInfo) Sys() interface{}  { return nil }

// resolveSymlinks resolves all symlinks in the given path, including the final component.
func resolveSymlinks(p string) (string, error) {
	return resolveSymlinksN(p, maxSymlinkDepth)
}

// resolveParentSymlinks resolves symlinks in all components except the final one.
// Used for lstat, readlink, unlink — operations that act on the symlink itself.
func resolveParentSymlinks(p string) (string, error) {
	parent := path.Dir(p)
	base := path.Base(p)
	if parent == "." || parent == "" {
		return p, nil
	}
	resolved, err := resolveSymlinks(parent)
	if err != nil {
		return "", err
	}
	if resolved == "." {
		return base, nil
	}
	return resolved + "/" + base, nil
}

func resolveSymlinksN(p string, limit int) (string, error) {
	if limit <= 0 {
		return "", &hackpadfs.PathError{Op: "lstat", Path: p, Err: syscall.ELOOP}
	}

	parts := strings.Split(p, "/")
	var processed []string

	for i, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(processed) > 0 {
				processed = processed[:len(processed)-1]
			}
			continue
		}

		processed = append(processed, part)
		current := strings.Join(processed, "/")

		target, isSymlink := lookupSymlink(current)
		if !isSymlink {
			continue
		}

		// Found a symlink at 'current'; resolve the target
		var targetPath string
		if strings.HasPrefix(target, "/") {
			// Absolute symlink: strip leading slash for hackpadfs paths
			targetPath = normalizeFsPath(target)
		} else {
			// Relative symlink: relative to the symlink's parent directory
			parent := strings.Join(processed[:len(processed)-1], "/")
			if parent == "" {
				targetPath = normalizeFsPath(target)
			} else {
				targetPath = normalizeFsPath(parent + "/" + target)
			}
		}

		// Append remaining path components after the symlink
		remaining := parts[i+1:]
		var fullPath string
		if len(remaining) > 0 {
			remainingStr := strings.Join(remaining, "/")
			if targetPath == "" || targetPath == "." {
				fullPath = remainingStr
			} else {
				fullPath = targetPath + "/" + remainingStr
			}
		} else {
			fullPath = targetPath
		}

		if fullPath == "" {
			fullPath = "."
		}

		return resolveSymlinksN(fullPath, limit-1)
	}

	result := strings.Join(processed, "/")
	if result == "" {
		return ".", nil
	}
	return result, nil
}

func normalizeFsPath(p string) string {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	if p == "" || p == "." {
		return "."
	}
	return p
}
