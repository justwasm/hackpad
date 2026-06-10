//go:build js

package fs

import (
	"github.com/hack-pad/hackpadfs/opfs"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

type persistFs struct {
	*opfs.OPFS
}

func newPersistDB(name string, relaxedDurability bool, shouldCache ShouldCacher) (*persistFs, error) {
	opfsFS, err := opfs.NewOPFS(name)
	return &persistFs{opfsFS}, err
}

func newBlobLength(i int) (blob.Blob, error) {
	return blob.NewBytesLength(i), nil
}
