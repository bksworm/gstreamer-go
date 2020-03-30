package msif

import (
	"bytes"
	"sync"
)

type BufferPool struct {
	p sync.Pool
}

func NewBufferPool(size int) (sp *BufferPool) {
	maker := func() interface{} {
		bb := new(bytes.Buffer)
		bb.Grow(size)
		return bb
	}
	sp = new(BufferPool)

	sp.p = sync.Pool{
		New: maker,
	}
	return sp
}

func (sp *BufferPool) Get() *bytes.Buffer {
	bb := sp.p.Get().(*bytes.Buffer)
	bb.Reset()
	return bb
}

func (sp *BufferPool) Put(bb *bytes.Buffer) {
	sp.p.Put(bb)
}
