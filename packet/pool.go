package packet

import (
	"bytes"
	"sync"
)

type Buffer struct {
	pool *sync.Pool
}

func newBuffer() *Buffer {
	return &Buffer{
		pool: &sync.Pool{
			New: func() any { return new(bytes.Buffer) },
		},
	}
}

func (b *Buffer) Get() *bytes.Buffer {
	return b.pool.Get().(*bytes.Buffer)
}

func (b *Buffer) Put(buf *bytes.Buffer) {
	buf.Reset()
	b.pool.Put(buf)
}

var buffer = newBuffer()

func GetBuffer() *bytes.Buffer {
	return buffer.Get()
}

func PutBuffer(buf *bytes.Buffer) {
	buffer.Put(buf)
}
