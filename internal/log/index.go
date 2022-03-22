package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

var (
	OffWidth uint64 = 4
	PosWidth uint64 = 8
	EntWidth uint64 = OffWidth + PosWidth
)

type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

func NewIndex(file *os.File, c Config) (*index, error) {
	idx := &index{file: file}
	fi, err := os.Stat(file.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(fi.Size())
	if err = os.Truncate(
		file.Name(), int64(c.Segment.MaxIndexBytes),
	); err != nil {
		return nil, err
	}
	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}

	return idx, nil
}

func (idx *index) Close() error {
	if err := idx.mmap.Sync(gommap.MS_ASYNC); err != nil {
		return err
	}
	if err := idx.file.Sync(); err != nil {
		return err
	}
	if err := idx.file.Truncate(int64(idx.size)); err != nil {
		return err
	}
	return idx.file.Close()
}

func (idx *index) Read(in int64) (out uint32, pos uint64, err error) {
	if idx.size == 0 {
		return 0, 0, io.EOF
	}
	if idx.size != 0 && in == -1 {
		out = uint32((idx.size / EntWidth) - 1)
	} else {
		out = uint32(in)
	}
	pos = uint64(out) * EntWidth
	if idx.size < pos+EntWidth {
		return 0, 0, io.EOF
	}
	out = Enc.Uint32(idx.mmap[pos : pos+OffWidth])
	pos = Enc.Uint64(idx.mmap[pos+OffWidth : pos+EntWidth])

	return out, pos, nil
}

func (idx *index) Write(off uint32, pos uint64) error {
	if uint64(len(idx.mmap)) < idx.size+EntWidth {
		return io.EOF
	}
	Enc.PutUint32(idx.mmap[idx.size:idx.size+OffWidth], off)
	Enc.PutUint64(idx.mmap[idx.size+OffWidth:idx.size+EntWidth], pos)
	idx.size += uint64(EntWidth)
	return nil
}

func (idx *index) Name() string {
	return idx.file.Name()
}

func (idx *index) Size() uint64 {
	return uint64(idx.size)
}
